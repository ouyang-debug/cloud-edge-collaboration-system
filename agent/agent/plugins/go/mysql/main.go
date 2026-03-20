//go:build plugin

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"agent/plus"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type input struct {
	TaskId     string `json:"taskId"`
	ConfigPath string `json:"configPath"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Pass       string `json:"pass"`
	DB         string `json:"db"`
	Query      string `json:"query"`
}

type yamlConn struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
	Timeout  int    `yaml:"timeout"`
	SslMode  string `yaml:"sslMode"`
}

type yamlExecute struct {
	TransactionMode string   `yaml:"transactionMode"`
	OnError         string   `yaml:"onError"`
	SqlList         []string `yaml:"sqlList"`
}

type yamlTask struct {
	Type       string      `yaml:"type"`
	TaskId     string      `yaml:"taskId"`
	Connection yamlConn    `yaml:"connection"`
	Execute    yamlExecute `yaml:"execute"`
}

type stmtResult struct {
	Index           int                    `json:"index"`
	Success         bool                   `json:"success"`
	RowsAffected    int64                  `json:"rowsAffected,omitempty"`
	Effective       bool                   `json:"effective,omitempty"`
	ExecutionTimeMs int64                  `json:"executionTimeMs"`
	Result          map[string]interface{} `json:"result,omitempty"`
	Error           map[string]string      `json:"error,omitempty"`
}

func pick(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func executeMySQLTask(in input, reporter plus.ProgressReporter) (map[string]interface{}, error) {
	var yt yamlTask
	if in.ConfigPath != "" {
		data, err := ioutil.ReadFile(in.ConfigPath)
		if err == nil {
			_ = yaml.Unmarshal(data, &yt)
		}
	}
	if yt.Connection.Host == "" && in.Host == "" {
		return buildFinal(in.TaskId, "mysql", "", "", false, false, nil, fmt.Errorf("empty host")), nil
	}
	host := pick(yt.Connection.Host, in.Host)
	port := in.Port
	if yt.Connection.Port > 0 {
		port = fmt.Sprintf("%d", yt.Connection.Port)
	}
	user := pick(yt.Connection.Username, in.User)
	pass := pick(yt.Connection.Password, in.Pass)
	db := pick(yt.Connection.Database, in.DB)
	charset := yt.Connection.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	timeoutSec := yt.Connection.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=%s&timeout=%ds", user, pass, host, port, db, charset, timeoutSec)
	mode := strings.ToLower(yt.Execute.TransactionMode)
	onError := strings.ToLower(yt.Execute.OnError)
	sqls := yt.Execute.SqlList
	if len(sqls) == 0 && in.Query != "" {
		sqls = []string{in.Query}
	}
	total := len(sqls)
	dbh, err := sql.Open("mysql", dsn)
	if err != nil {
		return buildFinal(in.TaskId, "mysql", "", mode, false, false, nil, err), err
	}
	defer dbh.Close()
	version := ""
	vr, _ := dbh.Query("SELECT VERSION()")
	if vr != nil {
		var v string
		if vr.Next() {
			_ = vr.Scan(&v)
			version = v
		}
		_ = vr.Close()
	}
	var results []stmtResult
	success := true
	committed := false
	if mode == "single" {
		tx, err := dbh.Begin()
		if err != nil {
			return buildFinal(in.TaskId, "mysql", version, mode, committed, false, nil, err), err
		}
		for i, s := range sqls {
			start := time.Now()
			r := execStmt(tx, s)
			r.Index = i
			r.ExecutionTimeMs = time.Since(start).Milliseconds()
			results = append(results, r)
			if reporter != nil && total > 0 {
				reporter.OnProgress(in.TaskId, "mysql", i+1, total, "")
			}
			if !r.Success {
				success = false
				if onError == "stop" {
					break
				}
			}
		}
		if success {
			if err := tx.Commit(); err == nil {
				committed = true
			}
		} else {
			_ = tx.Rollback()
		}
		return buildFinal(in.TaskId, "mysql", version, mode, committed, success, results, nil), nil
	}
	if mode == "multiple" {
		for i, s := range sqls {
			tx, err := dbh.Begin()
			if err != nil {
				return buildFinal(in.TaskId, "mysql", version, mode, false, false, results, err), err
			}
			start := time.Now()
			r := execStmt(tx, s)
			r.Index = i
			r.ExecutionTimeMs = time.Since(start).Milliseconds()
			if r.Success {
				_ = tx.Commit()
			} else {
				_ = tx.Rollback()
				success = false
				if onError == "stop" {
					results = append(results, r)
					break
				}
			}
			results = append(results, r)
			if reporter != nil && total > 0 {
				reporter.OnProgress(in.TaskId, "mysql", i+1, total, "")
			}
		}
		return buildFinal(in.TaskId, "mysql", version, mode, false, success, results, nil), nil
	}
	for i, s := range sqls {
		start := time.Now()
		r := execStmt(nil, s, dbh)
		r.Index = i
		r.ExecutionTimeMs = time.Since(start).Milliseconds()
		if !r.Success {
			success = false
			results = append(results, r)
			if onError == "stop" {
				if reporter != nil && total > 0 {
					reporter.OnProgress(in.TaskId, "mysql", i+1, total, "")
				}
				break
			}
			continue
		}
		results = append(results, r)
		if reporter != nil && total > 0 {
			reporter.OnProgress(in.TaskId, "mysql", i+1, total, "")
		}
	}
	return buildFinal(in.TaskId, "mysql", version, "none", false, success, results, nil), nil
}

func execStmt(tx *sql.Tx, s string, dbOpt ...*sql.DB) stmtResult {
	q := strings.TrimSpace(strings.ToLower(s))
	isQuery := strings.HasPrefix(q, "select")
	if isQuery {
		var rows *sql.Rows
		var err error
		if tx != nil {
			rows, err = tx.Query(s)
		} else {
			rows, err = dbOpt[0].Query(s)
		}
		if err != nil {
			return errStmt(err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		var outRows []map[string]interface{}
		count := 0
		for rows.Next() {
			if count >= 1000 {
				break
			}
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			_ = rows.Scan(ptrs...)
			row := make(map[string]interface{})
			for i, c := range cols {
				v := vals[i]
				switch t := v.(type) {
				case []byte:
					row[c] = string(t)
				case nil:
					row[c] = nil
				default:
					row[c] = t
				}
			}
			outRows = append(outRows, row)
			count++
		}
		return stmtResult{
			Success:         true,
			ExecutionTimeMs: 0,
			Result: map[string]interface{}{
				"rowCount": len(outRows),
				"rows":     outRows,
			},
		}
	}
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.Exec(s)
	} else {
		res, err = dbOpt[0].Exec(s)
	}
	if err != nil {
		return errStmt(err)
	}
	aff, _ := res.RowsAffected()
	return stmtResult{
		Success:         true,
		RowsAffected:    aff,
		Effective:       aff > 0,
		ExecutionTimeMs: 0,
	}
}

func errStmt(err error) stmtResult {
	m := map[string]string{
		"message": err.Error(),
	}
	return stmtResult{
		Success: false,
		Error:   m,
	}
}

func buildFinal(taskId string, dbType string, version string, mode string, committed bool, success bool, statements []stmtResult, err error) map[string]interface{} {
	out := map[string]interface{}{
		"taskId": taskId,
		"database": map[string]string{
			"type":    dbType,
			"version": version,
		},
		"transaction": map[string]interface{}{
			"mode":      mode,
			"committed": committed,
		},
		"success": success,
	}
	if err != nil {
		out["error"] = map[string]string{
			"message": err.Error(),
		}
	}
	if len(statements) > 0 {
		for i := range statements {
			if statements[i].Result == nil && statements[i].RowsAffected == 0 && statements[i].Error == nil {
				statements[i].Effective = false
			}
		}
		out["statements"] = statements
	}
	return out
}

type mysqlPlugin struct{}

func (m *mysqlPlugin) Name() string {
	return "mysql"
}

func (m *mysqlPlugin) Version() string {
	return "0.1.0"
}

func (m *mysqlPlugin) Description() string {
	return "MySQL plugin"
}

func (m *mysqlPlugin) Initialize(config string) error {
	return nil
}

func (m *mysqlPlugin) Execute(inputMap map[string]string) (map[string]string, error) {
	in := input{
		TaskId:     inputMap["taskId"],
		ConfigPath: inputMap["configPath"],
		Host:       inputMap["host"],
		Port:       inputMap["port"],
		User:       inputMap["user"],
		Pass:       inputMap["pass"],
		DB:         inputMap["db"],
		Query:      inputMap["query"],
	}
	out, err := executeMySQLTask(in, nil)
	if err != nil && out == nil {
		return nil, err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	result := map[string]string{
		"stdout": string(data),
	}
	if err != nil {
		result["stderr"] = err.Error()
	}
	return result, nil
}

func (m *mysqlPlugin) Shutdown() error {
	return nil
}

func (m *mysqlPlugin) ExecuteWithProgress(taskID string, inputMap map[string]string, reporter plus.ProgressReporter) (map[string]string, error) {
	in := input{
		TaskId:     inputMap["taskId"],
		ConfigPath: inputMap["configPath"],
		Host:       inputMap["host"],
		Port:       inputMap["port"],
		User:       inputMap["user"],
		Pass:       inputMap["pass"],
		DB:         inputMap["db"],
		Query:      inputMap["query"],
	}
	if in.TaskId == "" {
		in.TaskId = taskID
	}
	out, err := executeMySQLTask(in, reporter)
	//处理可能出现异常的情况
	if out != nil {
		if out["success"] == false {
			if statements, ok := out["statements"].([]stmtResult); ok {
				if len(statements) > 0 {
					// 访问第一个元素
					firstStatement := statements[0]
					// 检查是否有错误
					if !firstStatement.Success && firstStatement.Error != nil {
						// 处理错误
						err = fmt.Errorf("%s", firstStatement.Error["message"])
						return nil, err
					}
				}
			}
		}
	}
	if err != nil && out == nil {
		return nil, err
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	result := map[string]string{
		"stdout": string(data),
	}
	if err != nil {
		result["stderr"] = err.Error()
	}
	return result, nil
}

func (m *mysqlPlugin) OutputType() string {
	return "default"
}

func New() plus.Plugin {
	return &mysqlPlugin{}
}
