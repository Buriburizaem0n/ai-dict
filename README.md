# AI 词典 (AI Dictionary)

这是一个基于 Go 语言和大型语言模型 (LLM) API 构建的高性能、专业级实时在线词典。项目旨在提供快速、准确且内容丰富的单词查询体验，其核心能力完全由最先进的 AI 模型驱动。

![项目截图](https://storage.googleapis.com/gemini-prod-us-west1-d50e8804-6f2f/uploads/2024/08/23/project_screenshot.png)

## ✨ 功能特性 (Features)

* **实时 AI 生成**：所有词典数据均由 LLM 实时生成，内容鲜活且丰富。
* **专业级内容**：可提供包含音标、多词性、中英释义及情景例句的专业级词典内容。
* **极致性能优化**：
    * **毫秒级缓存**：内置内存缓存机制，重复查询的单词可实现瞬时响应。
    * **低 Token 消耗**：通过极限压缩的 Prompt Engineering，将单次查询的总 Token 消耗稳定在 200 以内，兼顾了速度与成本。
* **智能AI指令**：通过为 AI 注入“编辑判断力”，使其能够返回现代、实用的释义，并自动省略古老或罕见的用法，确保内容的专业性。
* **健壮的输入验证**：
    * 在前端和后端实施双重验证，限制输入内容的字符长度。
    * 有效防止长句或无效输入造成的资源浪费，并引导用户正确使用。
* **优雅的错误处理**：前端代码具备防御性编程能力，即使 AI 返回非预期格式，页面也不会崩溃。

## 🛠️ 技术栈 (Tech Stack)

* **后端 (Backend)**: Go (Golang) `net/http` 标准库
* **前端 (Frontend)**: HTML, CSS, Vanilla JavaScript
* **AI 服务**: 可通过 [OpenRouter](https://openrouter.ai/) 等平台调用任意大语言模型 API

## 🚀 快速开始 (Getting Started)

请按照以下步骤在您的本地计算机上运行本项目。

### 1. 先决条件 (Prerequisites)

确保您的系统已经安装了 [Go 语言](https://go.dev/doc/install) (建议版本 1.18 或以上)。

### 2. 安装与配置 (Installation & Configuration)

1.  **获取代码**:
    将项目文件（`main.go`, `static/` 文件夹等）放置在您选择的目录中。项目结构如下：
    ```
    /ai-dictionary
    ├── static/
    │   ├── index.html
    │   └── app.js
    └── main.go
    ```

2.  **获取并设置 API 密钥 (最关键的一步)**:
    * 访问 [OpenRouter](https://openrouter.ai/keys) 或其他您选择的 AI 服务商，获取您的 API Key。
    * 在main.go中配置url,Api-key以及选用的模型。
    * **提示**: 以上设置仅在当前终端会话中有效。要使其永久生效，请将命令添加到您的 shell 配置文件中 (如 `.zshrc` 或 `.bash_profile`)。

### 3. 运行项目 (Running the Application)

1.  打开终端，使用 `cd` 命令进入项目根目录 (`/ai-dictionary`)。
2.  运行以下命令来启动 Go 服务器：
    ```bash
    go run main.go
    ```
3.  如果一切顺利，您会看到提示 `Server starting on http://localhost:8080`。
4.  打开您的浏览器，访问 [http://localhost:8080](http://localhost:8080) 即可开始使用！

## ⚙️ 核心设计 (Core Design)

本项目成功的关键在于**精准的 Prompt Engineering**和**极致的性能优化**。

* **Prompt 设计**: 我们通过多次迭代，将最初近 300 token 的复杂指令，压缩为约 60 token 的高效指令。通过赋予 AI “词典编辑”的角色，并使用明确的排除指令 (`OMIT archaic...`)，我们引导模型在保持简洁的同时，输出高度专业和准确的内容。

* **性能优化**: 通过引入内存缓存，避免了对常用词的重复 API 调用，将响应时间从秒级降低到毫秒级。同时，前端和后端的双重输入验证有效保护了后端的 AI API 不被滥用。

## 🔮 未来展望 (Future Improvements)

* **流式响应 (Streaming)**: 实现打字机效果，进一步提升用户感知速度。
* **持久化缓存**: 使用 Redis 等工具替代内存缓存，使缓存数据在服务器重启后依然有效。
* **用户历史记录**: 增加用户查询历史的功能。
* **单词发音**: 利用文本转语音 (TTS) API，增加点击音标即可发音的功能。
* **混合词典模式**: 预先计算 2万个最常用单词并存入本地数据库，实现绝大多数查询的“绝对零延迟”。