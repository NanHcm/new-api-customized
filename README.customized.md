# new-api 定制版

[QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的个人定制 fork,主要解决几个使用订阅类 AI 服务时遇到的痛点。

完整功能、文档、UI 与上游保持一致,详见原项目 [README.md](./README.md)。

---

## 改了什么

### 1. 新增 `OpenAI Compatible`(OpenAI 兼容)渠道类型(Type 59)

**问题**:上游 new-api 的 `OpenAI` 类型会强制在 base URL 后拼接 `/v1/chat/completions`,导致非标准 OpenAI 兼容端点(国内厂商常用 `/v3`、`/v4`、`/paas/v4` 等路径)全部 404。`one-api` 有专门的 "OpenAI 兼容" 类型会自动剥掉 `/v1` 前缀,但 new-api fork 时丢失了这段逻辑。

**改动**:新增渠道类型 `OpenAI Compatible`,完全对标 `one-api` 的同名类型 — 用户填的 base URL 当作权威根地址,OpenAI 请求路径里的 `/v1` 段会被自动剥掉再拼接。

**用法对比**:

| 上游服务 | 之前(Custom + 完整 URL) | 现在(OpenAI Compatible + 短 URL) |
|---------|------------------------|----------------------------------|
| 智谱 GLM Coding Plan | `https://open.bigmodel.cn/api/coding/paas/v4/chat/completions` | `https://open.bigmodel.cn/api/coding/paas/v4` |
| 火山引擎 Agent Plan | `https://ark.cn-beijing.volces.com/api/plan/v3/chat/completions` | `https://ark.cn-beijing.volces.com/api/plan/v3` |
| 火山引擎 Coding Plan | `https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions` | `https://ark.cn-beijing.volces.com/api/coding/v3` |
| 任意非标准 OpenAI 兼容端点 | 必须填到 `chat/completions` | 填到根即可 |

涉及代码:
- `constant/channel.go` — 注册新常量 `ChannelTypeOpenAICompatible = 59`
- `common/api_type.go` — 复用 OpenAI adaptor
- `relay/common/relay_utils.go` — `GetFullRequestURL` 为 type 59 剥 `/v1`
- `relay/channel/openai/adaptor.go` — Claude/Gemini 分支同样剥 `/v1`
- `controller/channel_upstream_update.go` — fetch_models 用 `baseURL + /models`
- 前端 `web/src/features/channels/` — 渠道编辑器新增选项、字段透传、提示文案
- 7 个 i18n locale — 新增渠道类型说明翻译

### 2. fetch_models 失败时返回更明确的错误

**问题**:对于不提供 `/models` 端点的订阅计划(典型如火山引擎 Agent Plan),点击「获取模型」只显示模糊的 `status code: 404`,用户不知道是 key 错、URL 错、还是上游根本没这个接口。

**改动**:404 时附带中文说明 — `status code: 404 (上游未提供 /models 接口,请手动填写模型列表)`,告诉用户这就是上游限制,需要在「模型」输入框里手填逗号分隔的模型名。

涉及代码:`controller/channel_upstream_update.go::getFetchModelsResponseBody`

> 设计原则:**不伪造数据**。如果上游没有 `/models` 接口,就如实返回错误,而不是返回硬编码的"推荐列表"误导用户。

### 3. 移除原项目的 GitHub Actions workflows

`.github/workflows/*.yml` 是上游 QuantumNous 自身的 CI 配置(docker/electron/release 构建),引用了组织级 secrets,与个人 fork 无关,直接删除避免误触发。

---

## 完整改动列表

```
2a2d804 docs: 添加 README.customized.md 描述 fork 改动
1a10da3 chore: 删除个人 fork 不需要的 .github/workflows
31e53aa fix(channels): 上游无 /models 接口时返回更明确的错误
f9aad4f Revert: 不采用 fallback 推荐列表方案(保留真实错误)
92b47cc feat: 改进 fetch_models 错误提示(已被 revert,改用方案 2)
2d0b63e feat(channels): 新增 OpenAI Compatible 渠道类型(type 59)
```

涉及 18 个文件,+166 行。

---

## 验证测试

- `relay/common/relay_utils_test.go` — 新增 7 个单元测试,覆盖剥 `/v1` 的各种边界(尾斜杠、空格、embeddings 路径、无 `/v1` 前缀等)以及旧 OpenAI 类型的回归保护
- `controller/channel_upstream_update_test.go` — 已有测试全部通过
- OpenWrt 上实测:
  - GLM Coding Plan + OpenAI Compatible 类型 + 短 URL ✅ 正常对话
  - VolcEngine Agent Plan + OpenAI Compatible 类型 + 短 URL ✅ 正常对话
  - GLM fetch_models ✅ 返回 8 个真实模型
  - VolcEngine fetch_models ✅ 返回明确中文错误提示

---

## 部署

跟上游一致,只是 Docker image tag 不同。

### 用我构建的镜像

```bash
docker pull calciumion/new-api:openai-compat-v3
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/new-api:openai-compat-v3
```

### 自己构建

```bash
git clone https://github.com/NanHcm/new-api-customized.git
cd new-api-customized
docker buildx build --platform linux/amd64 -t new-api-customized:latest --load .
```

---

## 使用建议(订阅类渠道填法速查)

| 平台 / 订阅 | 渠道类型 | Base URL | 备注 |
|------------|---------|----------|------|
| 智谱 GLM Coding Plan(国内) | OpenAI Compatible | `https://open.bigmodel.cn/api/coding/paas/v4` | 也可用 Zhipu V4 + 魔法键 `glm-coding-plan` |
| 智谱 GLM Coding Plan(国际版 z.ai) | OpenAI Compatible | `https://api.z.ai/api/coding/paas/v4` | 也可用 `glm-coding-plan-international` |
| 火山引擎 Agent Plan | OpenAI Compatible | `https://ark.cn-beijing.volces.com/api/plan/v3` | 上游无 /models 接口,fetch_models 会报 404,需手填 |
| 火山引擎 Coding Plan | OpenAI Compatible | `https://ark.cn-beijing.volces.com/api/coding/v3` | 也可用 VolcEngine + 魔法键 `doubao-coding-plan` |
| Kimi Coding Plan | OpenAI Compatible | `https://api.kimi.com/coding/v1` | 也可用 Moonshot + 魔法键 `kimi-coding-plan` |
| 标准 OpenAI / DeepSeek / OpenRouter 等 | OpenAI | 默认值即可 | 不要选 OpenAI Compatible |
| 自建 LiteLLM 等中间层 | Custom | 完整 URL | 已支持 `{model}` 占位符 |

---

## 同步上游

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git
git fetch upstream
git merge upstream/main
# 解决可能的冲突(主要在 constant/channel.go 和前端渠道列表)
git push origin main
```

---

## 致谢与协议

本项目是 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的个人定制 fork,继承 AGPLv3 协议。原项目又基于 [songquanpeng/one-api](https://github.com/songquanpeng/one-api)(MIT License)。

- 完整原项目文档见 [README.md](./README.md)
- 协议见 [LICENSE](./LICENSE)
- 第三方依赖许可见 [THIRD-PARTY-LICENSES.md](./THIRD-PARTY-LICENSES.md)

Frontend design and development by New API contributors.
