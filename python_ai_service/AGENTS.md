# Python 检索服务

遵循 [根指南](../AGENTS.md)。服务独立运行，尚未接入聊天前端/backend、Compose 或 CI。

- app/main.py：FastAPI 路由、从搜索结果抽取句子的 answer。
- app/schemas.py：Pydantic 输入输出；config.py：基于源文件位置的 data/store.json。
- app/storage.py：进程内字典/Lock/TF-IDF/JSON，import 时 index.load()。

当前使用词法 TF-IDF，没有 LLM/embeddings；单进程文件存储不支持跨 worker 共享内存/锁。运行与 API 见 [README](README.md)，Docker 声明 Python 3.11。使用本地 .venv，不提交数据和缓存。

根目录 `make check-python` 只做语法检查。目前无自动测试；存储/检索改动使用临时数据路径验证空库、重载、同 ID 覆盖、删除和纯停用词输入，避免污染实际 store。Docker COPY data 需要构建上下文存在 data 目录，首次构建见 README。
