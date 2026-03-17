# Langpack 语言包功能

> 实现日期: 2026-03-17
> 服务路径: `app/bff/langpack/`

## 概述

实现了 Telegram MTProto 的 5 个 `langpack.*` 接口，为客户端提供多语言翻译支持。翻译数据首次访问时从 Telegram 官方翻译平台 (`translations.telegram.org`) 抓取，缓存在服务器本地，后续请求直接从本地返回。

## 支持的语言 (33 种)

| 语言码 | 语言名 | 原生名 |
|-------|--------|--------|
| en | English | English |
| ar | Arabic | العربية |
| be | Belarusian | Беларуская |
| ca | Catalan | Català |
| hr | Croatian | Hrvatski |
| cs | Czech | Čeština |
| nl | Dutch | Nederlands |
| fi | Finnish | Suomi |
| fr | French | Français |
| de | German | Deutsch |
| he | Hebrew | עברית |
| hu | Hungarian | Magyar |
| id | Indonesian | Bahasa Indonesia |
| it | Italian | Italiano |
| kk | Kazakh | Қазақша |
| ko | Korean | 한국어 |
| ms | Malay | Bahasa Melayu |
| nb | Norwegian | Norsk (Bokmål) |
| fa | Persian | فارسی |
| pl | Polish | Polski |
| pt-br | Portuguese (Brazil) | Português (Brasil) |
| ro | Romanian | Română |
| ru | Russian | Русский |
| sr | Serbian | Српски |
| sk | Slovak | Slovenčina |
| es | Spanish | Español |
| sv | Swedish | Svenska |
| tr | Turkish | Türkçe |
| uk | Ukrainian | Українська |
| uz | Uzbek | Oʻzbek |
| vi | Vietnamese | Tiếng Việt |
| **zh-hans** | **Chinese (Simplified)** | **简体中文** |
| **zh-hant** | **Chinese (Traditional)** | **繁體中文** |

## 5 个 API 接口

### 1. `langpack.getLanguages`
- **用途**: 客户端启动时获取可用语言列表
- **请求**: `langPack: string` (平台标识，如 `ios`、`android`)
- **返回**: `Vector<LangPackLanguage>` — 33 种语言的完整列表
- **数据来源**: 硬编码在 `dao/languages.go`

### 2. `langpack.getLanguage`
- **用途**: 获取单个语言的详细信息
- **请求**: `langPack: string, langCode: string`
- **返回**: `LangPackLanguage` — 语言名称、原生名、翻译数量等
- **错误**: 不支持的语言码返回 `LANG_CODE_NOT_SUPPORTED`

### 3. `langpack.getLangPack`
- **用途**: 获取某种语言的完整翻译包（通常 ~11000 条翻译）
- **请求**: `langPack: string, langCode: string`
- **返回**: `LangPackDifference` — 包含所有 `LangPackString` 条目
- **流程**:
  1. 查内存缓存 → 如有直接返回
  2. 查本地文件 `data/langpack/{platform}/{langCode}.strings` → 如有加载并返回
  3. 从 `https://translations.telegram.org/{langCode}/{platform}/export` 抓取
  4. 存储到本地文件 + 内存缓存 → 返回

### 4. `langpack.getStrings`
- **用途**: 获取指定 key 的翻译（如 `Login.ContinueWithLocalization`）
- **请求**: `langPack: string, langCode: string, keys: Vector<string>`
- **返回**: `Vector<LangPackString>` — 只返回匹配的 key
- **说明**: 内部调用 `getLangPack` 获取完整包再过滤

### 5. `langpack.getDifference`
- **用途**: 增量更新翻译（客户端提供 fromVersion）
- **请求**: `langPack: string, langCode: string, fromVersion: int`
- **返回**: `LangPackDifference`
- **当前行为**: 如果 fromVersion >= 当前版本返回空 diff，否则返回全量

## 文件结构

```
app/bff/langpack/
├── helper.go                              # 对外入口 New() + Config
└── internal/
    ├── config/config.go                   # zrpc.RpcServerConf
    ├── svc/service_context.go             # ServiceContext = Config + Dao
    ├── dao/
    │   ├── dao.go                         # 核心: 抓取 + 解析 + 缓存
    │   └── languages.go                   # 33 种语言定义
    ├── core/
    │   ├── core.go                        # LangpackCore 请求上下文
    │   ├── langpack.getLanguages_handler.go
    │   ├── langpack.getLanguage_handler.go
    │   ├── langpack.getLangPack_handler.go
    │   ├── langpack.getStrings_handler.go
    │   └── langpack.getDifference_handler.go
    └── server/grpc/
        ├── grpc.go                        # RegisterRPCLangpackServer
        └── service/
            ├── service.go
            └── langpack_service_impl.go
```

## 缓存策略

```
请求 → [内存缓存] → [本地文件] → [Telegram API]
                                       ↓
                                 保存到本地文件
                                       ↓
                                 加载到内存缓存
```

- **内存缓存**: `map[string]*LangPackEntry`，key 格式 `{platform}/{langCode}`，进程生命周期内有效
- **本地文件**: `data/langpack/{platform}/{langCode}.strings`，Apple `.strings` 格式
- **首次抓取**: 从 `translations.telegram.org` HTTP GET，约 200KB/语言
- **后续请求**: 直接从内存返回，零延迟

## 平台映射

| 客户端 lang_pack | Telegram 平台 |
|-----------------|--------------|
| `ios`, `macos` | `ios` |
| `android`, `android_x` | `android` |
| `tdesktop`, `desktop` | `tdesktop` |
| 其他/空 | `ios` (默认) |

## 配置变更

### session.yaml (两个文件)
```yaml
# 取消注释:
"/mtproto.RPCLangpack": "bff.bff"
```

### BFF server.go
```go
// 新增注册:
mtproto.RegisterRPCLangpackServer(grpcServer, langpack_helper.New(...))
```

### fake_rpc_result.go
移除了 `TLLangpackGetDifference`、`TLLangpackGetLangPack`、`TLLangpackGetLanguages`、`TLLangpackGetStrings` 四个 fake 返回。

## iOS 客户端请求流程参考

根据 Telegram iOS 客户端日志 (`Telegram_lang.rtf`)：

1. **启动时** 同时发送:
   - `langpack.getLanguages(langPack: "")` — 获取语言列表
   - `langpack.getStrings(langPack: "", langCode: "en", keys: ["Login.ContinueWithLocalization"])` — 登录页按钮文字

2. **用户切换语言时**:
   - `langpack.getLanguage(langPack: "", langCode: "ms")` — 获取目标语言信息
   - `langpack.getLangPack(langPack: "", langCode: "ms")` — 获取完整翻译包

3. **后续启动**:
   - `langpack.getDifference(langPack: "", langCode: "ms", fromVersion: 20456364)` — 增量更新

## 注意事项

1. **网络依赖**: 首次获取某语言时需要服务器能访问 `translations.telegram.org`。如果服务器在中国大陆，可能需要代理。
2. **无数据库**: 当前实现不使用数据库，完全基于文件缓存。进程重启后内存缓存会丢失，但文件缓存仍在。
3. **版本号**: 当前使用固定版本号 1，不支持真正的增量更新。客户端版本不匹配时会收到全量响应。
4. **认证**: 所有 5 个方法已在 `check_api_request_type.go` 中白名单，无需登录即可调用。
