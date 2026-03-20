package plus

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"sync"
)

type ProgressReporter interface {
	// 当插件执行有阶段进度时调用
	// 参数含义：
	// - taskID: 任务标识
	// - pluginName: 插件名称
	// - current/total: 当前进度与总进度，用于计算百分比
	// - message: 进度说明或提示信息
	OnProgress(taskID, pluginName string, current, total int, message string)
	// 当插件执行完成时调用
	// 参数含义：
	// - success: 是否执行成功
	// - message: 完成说明或提示信息
	OnCompleted(taskID, pluginName string, success bool, message string)
	// 当插件执行发生错误时调用
	// 参数含义：
	// - err: 错误对象，需包含可读的错误信息
	OnError(taskID, pluginName string, err error)
}

type Plugin interface {
	// 返回插件唯一名称（用于注册、按名称加载与执行）
	Name() string
	// 返回插件版本号（可用于兼容性检查）
	Version() string
	// 返回插件功能与用途的简要描述
	Description() string
	// 返回插件输出类型，默认是txt，可选json、monitor
	OutputType() string
	// 初始化插件，传入配置路径或空字符串，进行资源准备
	// 常见用法：解析配置、建立连接、加载资源等
	Initialize(config string) error
	// 执行插件的主要逻辑，不带进度上报；返回键值结果
	// 入参 input：任务输入的键值对，例如 cmd、configPath 等
	// 返回 map：执行结果的键值对，例如 stdout、stderr、artifacts 等
	Execute(input map[string]string) (map[string]string, error)
	// 带进度上报的执行，适用于多阶段任务
	// 参数含义：
	// - taskID: 任务标识
	// - input: 任务输入键值对
	// - reporter: 进度上报器（调用 OnProgress/OnCompleted/OnError）
	ExecuteWithProgress(taskID string, input map[string]string, reporter ProgressReporter) (map[string]string, error)
	// 插件卸载前清理资源（关闭连接、释放文件句柄等）
	Shutdown() error
}

// PluginInfo represents information about a plugin
type PluginInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	OutputType  string `json:"outputType"`
	Description string `json:"description"`
	IsDefault   bool   `json:"isDefault"`
}

// PluginManager manages plugins and their execution
type PluginManager struct {
	plugins         map[string]map[string]Plugin // map[pluginName]map[version]Plugin
	defaultVersions map[string]string            // map[pluginName]defaultVersion
	mutex           sync.RWMutex
}

// NewPluginManager creates a new PluginManager instance
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:         make(map[string]map[string]Plugin),
		defaultVersions: make(map[string]string),
	}
}

// Start initializes and starts the PluginManager
func (pm *PluginManager) Start() error {
	log.Println("Starting Plugin Manager...")

	// Create plugins directory if it doesn't exist
	if err := pm.ensurePluginDirectory(); err != nil {
		return fmt.Errorf("failed to ensure plugin directory: %v", err)
	}

	// Load all plugins from the plugins directory
	if err := pm.loadPlugins(); err != nil {
		log.Printf("Warning: failed to load plugins: %v", err)
	}

	log.Println("Plugin Manager started")
	return nil
}

// Stop stops the PluginManager and all loaded plugins
func (pm *PluginManager) Stop() error {
	log.Println("Stopping Plugin Manager...")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Shutdown all plugins
	for name, versions := range pm.plugins {
		for version, plugin := range versions {
			if err := plugin.Shutdown(); err != nil {
				log.Printf("Error shutting down plugin %s version %s: %v", name, version, err)
			}
		}
		delete(pm.plugins, name)
		delete(pm.defaultVersions, name)
	}

	log.Println("Plugin Manager stopped")
	return nil
}

// LoadPlugin loads a plugin from the given path
func (pm *PluginManager) LoadPlugin(pluginPath string) (string, error) {
	log.Printf("Loading plugin from %s...", pluginPath)

	// Open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return "", fmt.Errorf("failed to open plugin %s: %v", pluginPath, err)
	}

	// Look up the New function
	newFunc, err := p.Lookup("New")
	if err != nil {
		return "", fmt.Errorf("failed to find New function in plugin %s: %v", pluginPath, err)
	}

	// Assert the type of New function
	pluginFactory, ok := newFunc.(func() Plugin)
	if !ok {
		return "", fmt.Errorf("invalid plugin factory type in %s", pluginPath)
	}

	// Create a new plugin instance
	pluginInstance := pluginFactory()

	// Initialize the plugin
	if err := pluginInstance.Initialize(""); err != nil {
		return "", fmt.Errorf("failed to initialize plugin %s: %v", pluginPath, err)
	}

	// Add the plugin to the manager
	pm.mutex.Lock()
	pluginName := pluginInstance.Name()
	pluginVersion := pluginInstance.Version()

	// Check if plugin name exists in the map
	if _, exists := pm.plugins[pluginName]; !exists {
		pm.plugins[pluginName] = make(map[string]Plugin)
		// Set the first version as default
		pm.defaultVersions[pluginName] = pluginVersion
	} else {
		// Check if the new version is higher than the current default
		currentDefault := pm.defaultVersions[pluginName]
		if compareVersions(pluginVersion, currentDefault) > 0 {
			// New version is higher, update default
			pm.defaultVersions[pluginName] = pluginVersion
			log.Printf("Updated default version for plugin %s to %s (higher version)", pluginName, pluginVersion)
		}
	}

	// Store the plugin instance by version
	pm.plugins[pluginName][pluginVersion] = pluginInstance
	pm.mutex.Unlock()

	log.Printf("Loaded plugin: %s (version %s)", pluginName, pluginVersion)
	return pluginName, nil
}

// UnloadPlugin unloads a plugin by name and version
// If version is empty, unloads all versions of the plugin
func (pm *PluginManager) UnloadPlugin(pluginName string, version string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pluginVersions, exists := pm.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if version == "" {
		// Unload all versions
		for v, pluginInstance := range pluginVersions {
			if err := pluginInstance.Shutdown(); err != nil {
				log.Printf("Error shutting down plugin %s version %s: %v", pluginName, v, err)
			}
		}
		delete(pm.plugins, pluginName)
		delete(pm.defaultVersions, pluginName)
		log.Printf("Unloaded all versions of plugin: %s", pluginName)
	} else {
		// Unload specific version
		pluginInstance, exists := pluginVersions[version]
		if !exists {
			return fmt.Errorf("plugin %s version %s not found", pluginName, version)
		}

		// Shutdown the plugin
		if err := pluginInstance.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown plugin %s version %s: %v", pluginName, version, err)
		}

		// Remove the version from the map
		delete(pluginVersions, version)

		// If no versions left, remove the plugin entirely
		if len(pluginVersions) == 0 {
			delete(pm.plugins, pluginName)
			delete(pm.defaultVersions, pluginName)
		} else if pm.defaultVersions[pluginName] == version {
			// If we're removing the default version, set a new default
			for v := range pluginVersions {
				pm.defaultVersions[pluginName] = v
				break
			}
		}

		log.Printf("Unloaded plugin: %s (version %s)", pluginName, version)
	}

	return nil
}

// GetPlugin gets a plugin by name and version
// If version is empty, uses the default version
func (pm *PluginManager) GetPlugin(pluginName string, version string) (Plugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pluginVersions, exists := pm.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	if version == "" {
		// Use default version
		defaultVersion, exists := pm.defaultVersions[pluginName]
		if !exists {
			return nil, fmt.Errorf("no default version set for plugin %s", pluginName)
		}
		version = defaultVersion
	}

	pluginInstance, exists := pluginVersions[version]
	if !exists {
		return nil, fmt.Errorf("plugin %s version %s not found", pluginName, version)
	}

	return pluginInstance, nil
}

func (pm *PluginManager) GetPluginOutputType(pluginName string, version string) (string, error) {
	pluginInstance, err := pm.GetPlugin(pluginName, version)
	if err != nil {
		return "", err
	}
	return pluginInstance.OutputType(), nil
}

// ExecutePlugin executes a plugin with the given input
// If version is empty, uses the default version
func (pm *PluginManager) ExecutePlugin(pluginName string, version string, input map[string]string) (map[string]string, error) {
	pluginInstance, err := pm.GetPlugin(pluginName, version)
	if err != nil {
		return nil, err
	}

	output, err := pluginInstance.Execute(input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin %s version %s: %v", pluginName, version, err)
	}

	return output, nil
}

// ExecutePluginWithProgress executes a plugin with progress reporting
// If version is empty, uses the default version
func (pm *PluginManager) ExecutePluginWithProgress(pluginName string, version string, taskID string, input map[string]string, reporter ProgressReporter) (map[string]string, error) {
	pluginInstance, err := pm.GetPlugin(pluginName, version)
	if err != nil {
		return nil, err
	}
	if reporter == nil {
		return pm.ExecutePlugin(pluginName, version, input)
	}
	output, err := pluginInstance.ExecuteWithProgress(taskID, input, reporter)
	if err != nil {
		reporter.OnError(taskID, pluginName, err)
		return nil, fmt.Errorf("failed to execute plugin %s version %s: %v", pluginName, version, err)
	}
	reporter.OnCompleted(taskID, pluginName, true, "")
	return output, nil
}

// ListPlugins returns a list of all loaded plugins with their versions
func (pm *PluginManager) ListPlugins() ([]PluginInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var plugins []PluginInfo
	for pluginName, versions := range pm.plugins {
		defaultVersion := pm.defaultVersions[pluginName]
		for version, plugin := range versions {
			plugins = append(plugins, PluginInfo{
				ID:          fmt.Sprintf("%s@%s", pluginName, version),
				Name:        pluginName,
				Version:     version,
				Path:        "", // Path is not tracked in the current implementation
				Status:      "loaded",
				Description: plugin.Description(),
				IsDefault:   version == defaultVersion,
			})
		}
	}

	return plugins, nil
}

// ensurePluginDirectory ensures that the plugins directory exists
func (pm *PluginManager) ensurePluginDirectory() error {
	pluginDir := filepath.Join(".", "plugins")
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %v", err)
		}
	}

	return nil
}

// GetPluginVersions returns all versions of a plugin
func (pm *PluginManager) GetPluginVersions(pluginName string) ([]string, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pluginVersions, exists := pm.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	var versions []string
	for version := range pluginVersions {
		versions = append(versions, version)
	}

	return versions, nil
}

// compareVersions compares two version strings and returns:
// 1 if v1 > v2
// 0 if v1 == v2
// -1 if v1 < v2
func compareVersions(v1, v2 string) int {
	// Simple semantic version comparison
	// This is a basic implementation, you may need to enhance it for complex version strings
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		var err1, err2 error

		if i < len(parts1) {
			num1, err1 = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			num2, err2 = strconv.Atoi(parts2[i])
		}

		// If conversion fails, treat as 0
		if err1 != nil {
			num1 = 0
		}
		if err2 != nil {
			num2 = 0
		}

		if num1 > num2 {
			return 1
		} else if num1 < num2 {
			return -1
		}
	}

	return 0
}

// SetDefaultVersion sets the default version for a plugin
// If version is empty, sets the highest version as default
func (pm *PluginManager) SetDefaultVersion(pluginName string, version string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pluginVersions, exists := pm.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if version != "" {
		// Set specific version
		if _, exists := pluginVersions[version]; !exists {
			return fmt.Errorf("plugin %s version %s not found", pluginName, version)
		}
		pm.defaultVersions[pluginName] = version
		log.Printf("Set default version for plugin %s to %s", pluginName, version)
	} else {
		// Set highest version as default
		highestVersion := ""
		for v := range pluginVersions {
			if highestVersion == "" || compareVersions(v, highestVersion) > 0 {
				highestVersion = v
			}
		}
		if highestVersion != "" {
			pm.defaultVersions[pluginName] = highestVersion
			log.Printf("Set default version for plugin %s to highest version %s", pluginName, highestVersion)
		}
	}

	return nil
}

// loadPlugins loads all plugins from the plugins directory
func (pm *PluginManager) loadPlugins() error {
	pluginDir := filepath.Join(".", "plugins")

	files, err := filepath.Glob(filepath.Join(pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to glob plugin directory: %v", err)
	}

	// Load each plugin
	for _, file := range files {
		if _, err := pm.LoadPlugin(file); err != nil {
			log.Printf("Warning: failed to load plugin %s: %v", file, err)
		}
	}

	return nil
}
