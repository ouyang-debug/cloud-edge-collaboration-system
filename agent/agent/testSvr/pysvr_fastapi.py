# coding=utf-8
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from typing import Optional
import base64

from fastapi import UploadFile, File, Form
from fastapi.responses import JSONResponse
import os
import hashlib
import aiofiles
# 初始化 FastAPI 应用
app = FastAPI(title="Log Data Receiver", version="1.0")

# 定义 Pydantic 模型（对应 LogDataRequest 结构体，自动校验数据格式）
class LogDataRequest(BaseModel):
    taskId: str = Field(..., description="任务ID")
    agentId: str = Field(..., description="agentID")
    logLevel: str = Field(..., description="日志级别")
    logType: str = Field(..., description="日志类型")
    content: str = Field(..., description="日志内容")
    read_time: str = Field(..., description="读取时间（建议ISO8601格式）")
    offset: int = Field(..., description="偏移量")
    size: int = Field(..., description="读取内容大小（字节）")
    content_hash: str = Field(..., description="内容哈希值")

class RegDataRequest(BaseModel):
    agent_id: str = Field(..., alias="agentId", description="agent")
    data: str = Field(..., description="注册信息")

    class Config:
        populate_by_name = True

class StatusDataRequest(BaseModel):
    client_id: str = Field(..., description="agent")
    task_id: str = Field(..., description="任务ID")
    status: str = Field(..., description="状态")
    message: Optional[str] = Field(None, description="状态消息")

class ResultSync(BaseModel):
    client_id: str = Field(..., description="agent")
    task_id: str = Field(..., description="任务ID")
    step: int = Field(..., description="步骤")
    file_content: str = Field(..., description="文件内容")
    create_at: str = Field(..., description="创建时间（建议ISO8601格式）")

# 定义 POST 接口：/logs
@app.post("/logs", summary="接收日志数据", response_description="返回接收结果")
async def receive_logs(log_data: LogDataRequest):
    # 1. 数据已通过 Pydantic 自动校验（格式、必填项、类型）
    # 2. 模拟业务处理（替换为你的实际逻辑，如存数据库、写日志等）
    content_bytes = base64.b64decode(log_data.content)
    encodings = ["utf-8", "gbk", "gb2312", "latin-1", "ascii"]
    content_decoded = None

    # 依次尝试解码
    for encoding in encodings:
        try:
            content_decoded = content_bytes.decode(encoding)
            print(f"成功使用 {encoding} 解码")
            break
        except UnicodeDecodeError:
            continue
    if content_decoded is None:
        raise HTTPException(
            status_code=400,
            detail="无法使用任何编码解码Content字段"
        )

    print("接收到的日志数据：")
    print(f"任务ID：{log_data.taskId}")
    print(f"agentID：{log_data.agentId}")
    print(f"日志级别：{log_data.logLevel}")
    print(f"日志类型：{log_data.logType}")
    print(f"日志内容base64：{log_data.content}")
    print(f"日志内容：{content_decoded}")
    print(f"读取时间：{log_data.read_time}")
    print(f"偏移量：{log_data.offset}")
    print(f"大小：{log_data.size}")
    print(f"内容哈希：{log_data.content_hash}")

    # 3. 返回成功响应
    return {
        "code": 200,
        "message": "日志数据接收成功",
        "data": log_data.dict()  # 将 Pydantic 模型转为字典返回
    }

# 定义 POST 接口：/api/env-node-registration/register
@app.post("/api/env-node-registration/register", summary="注册", response_description="返回注册结果")
async def receive_register(reg_data: RegDataRequest):
    print("接收到的注册数据：")
    print(f"agent：{reg_data.agent_id}")
    print(f"注册信息：{reg_data.data}")

    return {
        "code": "0",
        "msg": "注册成功"
    }


@app.post("/task-status", summary="状态", response_description="返回状态结果")
async def receive_status(status_data: StatusDataRequest):
    # 1. 数据已通过 Pydantic 自动校验（格式、必填项、类型）
    # 2. 模拟业务处理（替换为你的实际逻辑，如存数据库、写日志等）
    print("接收到的状态数据：")
    print(f"agent：{status_data.client_id}")
    print(f"任务ID：{status_data.task_id}")
    print(f"状态：{status_data.status}")  
    print(f"状态消息：{status_data.message}")  

    # 3. 返回成功响应
    return {
        "code": 0,
        "message": "状态成功"
    }



# 上传文件存储目录
UPLOAD_DIR = "./upload_files"
os.makedirs(UPLOAD_DIR, exist_ok=True)

@app.post("/api/upload/check", tags=["断点续传"])
async def check_uploaded_size(
    file_md5: str = Form(...),
    file_name: str = Form(...)
):
    """查询文件已上传的大小"""
    if not file_md5 or not file_name:
        raise HTTPException(status_code=400, detail="缺少文件标识")

    file_path = os.path.join(UPLOAD_DIR, file_name)
    if os.path.exists(file_path):
        uploaded_size = os.path.getsize(file_path)
        return JSONResponse(content={
            "code": 200,
            "uploaded_size": uploaded_size,
            "msg": "文件已存在部分分片"
        })
    else:
        return JSONResponse(content={
            "code": 200,
            "uploaded_size": 0,
            "msg": "文件未上传过"
        })

@app.post("/api/upload/continue", tags=["断点续传"])
async def continue_upload(
    file_md5: str = Form(...),
    file_name: str = Form(...),
    start_pos: int = Form(0),
    file_chunk: UploadFile = File(...)
):
    """接收分片数据，实现断点续传"""
    if not file_md5 or not file_name or not file_chunk:
        raise HTTPException(status_code=400, detail="参数缺失")

    file_path = os.path.join(UPLOAD_DIR, file_name)

    # 异步写入文件（追加模式，定位到断点位置）
    try:
        if os.path.exists(file_path):
            async with aiofiles.open(file_path, 'rb+') as f:
                await f.seek(start_pos)
                content = await file_chunk.read()
                await f.write(content)
        else:
            async with aiofiles.open(file_path, 'wb') as f:
                await f.seek(start_pos)
                content = await file_chunk.read()
                await f.write(content)

        current_size = os.path.getsize(file_path)
        return JSONResponse(content={
            "code": 200,
            "current_size": current_size,
            "msg": "分片上传成功"
        })
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"文件写入失败：{str(e)}")

# 文本结果回传：/result_sync
@app.post("/agent/result_sync", summary="结果同步", response_description="返回同步结果")
async def receive_result_sync(result_sync: ResultSync):
    # 1. 数据已通过 Pydantic 自动校验（格式、必填项、类型）
    # 2. 模拟业务处理（替换为你的实际逻辑，如存数据库、写日志等）
    print("接收到的结果同步数据：")
    print(f"agent：{result_sync.client_id}")
    print(f"任务ID：{result_sync.task_id}")
    print(f"步骤：{result_sync.step}")
    print(f"文件内容：{result_sync.file_content}")
    print(f"创建时间：{result_sync.create_at}")

    # 3. 返回成功响应
    return {
        "code": 0,
        "message": "结果同步成功"
    }

# 启动服务（直接运行该脚本即可）
if __name__ == "__main__":
    import uvicorn
    # 监听 8080 端口，允许外部访问，开启自动重载（debug模式）
    uvicorn.run(app, host="0.0.0.0", port=8081)#, reload=True)
