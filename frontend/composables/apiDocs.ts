export type ApiDocParameter = {
  name: string
  location: 'Header' | 'Query' | 'Path'
  required: boolean | 'conditional'
  type: string
  description: string
}

export type ApiDocField = {
  name: string
  description: string
}

export type ApiDocStatus = {
  code: number
  title: string
  description: string
  example: string
  fields: ApiDocField[]
}

export type ApiEndpointDoc = {
  slug: string
  navLabel: string
  name: string
  introduction: string
  method: 'GET'
  path: string
  auth: string
  parameters: ApiDocParameter[]
  requestExample: string
  statuses: ApiDocStatus[]
}

const tokenHeaderParameters: ApiDocParameter[] = [
  {
    name: 'Authorization',
    location: 'Header',
    required: 'conditional',
    type: 'Bearer token',
    description: '推荐方式。值为 `Bearer <tgal_live API token>`。与 `X-API-Token` 二选一。'
  },
  {
    name: 'X-API-Token',
    location: 'Header',
    required: 'conditional',
    type: 'string',
    description: '备选方式。与 Authorization 二选一；同时传入有效 Bearer header 时优先使用 Authorization。'
  }
]

const noParameterStatuses: ApiDocField[] = [
  { name: 'success', description: '固定为 true，表示请求成功。' },
  { name: 'data.status', description: '当前服务状态。' },
  { name: 'data.version', description: '当前公开 API 主版本。' }
]

const errorFields: ApiDocField[] = [
  { name: 'success', description: '固定为 false，表示请求失败。' },
  { name: 'error.code', description: '稳定错误码，可用于客户端分支处理。' },
  { name: 'error.message', description: '面向开发者的错误摘要，不包含内部 SQL、token、DSN 或其他敏感信息。' }
]

const unauthorizedStatus: ApiDocStatus = {
  code: 401,
  title: 'Unauthorized',
  description: '缺少 API token、token 格式错误、token 已删除，或所属应用/账号不可用。',
  example: `{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid API token"
  }
}`,
  fields: errorFields
}

const rateLimitedStatus: ApiDocStatus = {
  code: 429,
  title: 'Rate limited',
  description: '触发预认证 IP 限流，或触发 token、账号、应用三维之一的分钟/日限流。通过 token 认证后触发的限流响应会带 X-RateLimit-* 计数；预认证 IP 限流不会带这些响应头。',
  example: `{
  "success": false,
  "error": {
    "code": "RATE_LIMITED",
    "message": "API rate limit exceeded"
  }
}`,
  fields: [
    ...errorFields,
    { name: 'X-RateLimit-Limit-Minute', description: '通过 token 认证后返回；本次认证上下文下最紧的分钟额度。' },
    { name: 'X-RateLimit-Remaining-Minute', description: '通过 token 认证后返回；当前分钟窗口剩余额度。' },
    { name: 'X-RateLimit-Limit-Day', description: '通过 token 认证后返回；本次认证上下文下最紧的日额度。' },
    { name: 'X-RateLimit-Remaining-Day', description: '通过 token 认证后返回；当前日窗口剩余额度。' }
  ]
}

const internalStatus: ApiDocStatus = {
  code: 500,
  title: 'Internal error',
  description: '服务端内部错误或限流依赖异常。客户端应记录 request id 并稍后重试，不应把它当作参数错误处理。',
  example: `{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Internal server error"
  }
}`,
  fields: errorFields
}

export const apiEndpointDocs: ApiEndpointDoc[] = [
  {
    slug: 'health',
    navLabel: '健康检查',
    name: '健康检查',
    introduction: '返回 API 进程是否存活，不访问 PostgreSQL、Redis 或源数据库。适合负载均衡的轻量存活探针。',
    method: 'GET',
    path: '/v1/health',
    auth: '无需 API token。',
    parameters: [],
    requestExample: `curl "https://api.example.com/v1/health"`,
    statuses: [
      {
        code: 200,
        title: 'OK',
        description: 'API 进程已响应。该结果只证明 HTTP 进程存活，不代表数据库和 Redis 可用。',
        example: `{
  "success": true,
  "data": {
    "status": "ok",
    "version": "v1"
  }
}`,
        fields: noParameterStatuses
      }
    ]
  },
  {
    slug: 'ready',
    navLabel: '就绪检查',
    name: '就绪检查',
    introduction: '检查 clean PostgreSQL 与 Redis 依赖是否可用，不触碰 TouchGal 主库。适合部署平台的 readiness probe。',
    method: 'GET',
    path: '/v1/ready',
    auth: '无需 API token。',
    parameters: [],
    requestExample: `curl "https://api.example.com/v1/ready"`,
    statuses: [
      {
        code: 200,
        title: 'Ready',
        description: '所有已配置的 readiness checks 均通过。',
        example: `{
  "success": true,
  "data": {
    "status": "ready",
    "version": "v1"
  }
}`,
        fields: noParameterStatuses
      },
      {
        code: 503,
        title: 'Not ready',
        description: 'readiness checks 未配置、超时，或 clean PostgreSQL/Redis 当前不可用。',
        example: `{
  "success": false,
  "error": {
    "code": "NOT_READY",
    "message": "Service dependencies are not ready"
  }
}`,
        fields: errorFields
      }
    ]
  },
  {
    slug: 'token-self-check',
    navLabel: 'Token 自检',
    name: 'API token 自检',
    introduction: '验证当前 token，并返回该 token 所属应用与实际生效的请求额度。不会返回 token 明文或 token hash。',
    method: 'GET',
    path: '/v1/me',
    auth: '需要有效的 `tgal_live` API token。',
    parameters: tokenHeaderParameters,
    requestExample: `curl "https://api.example.com/v1/me" \\
  -H "Authorization: Bearer tgal_live_xxx"`,
    statuses: [
      {
        code: 200,
        title: 'OK',
        description: 'token 已通过认证，且所属账号/应用允许访问公开 API。',
        example: `{
  "success": true,
  "data": {
    "tokenPrefix": "tgal_live_abcd1234efgh5678ijkl90",
    "applicationId": "018f35d5-3a7f-7b90-8d72-173b6f94c3da",
    "applicationStatus": "approved",
    "minuteLimit": 60,
    "dailyLimit": 1000
  }
}`,
        fields: [
          { name: 'success', description: '固定为 true。' },
          { name: 'data.tokenPrefix', description: 'token 前缀，仅用于识别 token；不是可再次鉴权的 secret。' },
          { name: 'data.applicationId', description: 'token 绑定的开发者应用 UUID。' },
          { name: 'data.applicationStatus', description: '应用状态。公开 API 可用时通常为 approved。' },
          { name: 'data.minuteLimit', description: 'token、账号、应用三维综合后的有效分钟额度。' },
          { name: 'data.dailyLimit', description: 'token、账号、应用三维综合后的有效日额度。' }
        ]
      },
      unauthorizedStatus,
      rateLimitedStatus,
      internalStatus
    ]
  },
  {
    slug: 'games-search',
    navLabel: '搜索条目',
    name: '搜索游戏条目',
    introduction: '按关键词搜索 clean DB 中未删除且 SFW 的公开 Galgame 条目。',
    method: 'GET',
    path: '/v1/games/search',
    auth: '需要有效的 `tgal_live` API token。',
    parameters: [
      ...tokenHeaderParameters,
      {
        name: 'keyword',
        location: 'Query',
        required: 'conditional',
        type: 'string, 3-100 Unicode chars',
        description: '搜索关键词。`keyword` 与 `q` 二选一；优先读取 `keyword`。空白字符串会按无效参数处理。'
      },
      {
        name: 'q',
        location: 'Query',
        required: 'conditional',
        type: 'string, 3-100 Unicode chars',
        description: '`keyword` 的短别名。仅当 `keyword` 为空字符串时使用。'
      },
      {
        name: 'page',
        location: 'Query',
        required: false,
        type: 'integer, 1-100',
        description: '页码。默认 1；小于 1 或非整数按 1 处理；超过 100 返回 BAD_REQUEST。'
      },
      {
        name: 'limit',
        location: 'Query',
        required: false,
        type: 'integer, 1-50',
        description: '每页数量。默认 20；小于 1 或非整数按 20 处理；超过 50 会按 50 裁剪。'
      }
    ],
    requestExample: `curl "https://api.example.com/v1/games/search?keyword=summer&page=1&limit=10" \\
  -H "Authorization: Bearer tgal_live_xxx"`,
    statuses: [
      {
        code: 200,
        title: 'OK',
        description: '返回匹配条目列表与分页信息。搜索列表只包含名称与公开 uniqueId，详情请继续调用条目详情接口。',
        example: `{
  "success": true,
  "data": {
    "items": [
      {
        "name": "Summer Pockets",
        "uniqueId": "abcd1234"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1,
      "hasMore": false
    }
  }
}`,
        fields: [
          { name: 'success', description: '固定为 true。' },
          { name: 'data.items[].name', description: '公开条目名称。' },
          { name: 'data.items[].uniqueId', description: '8 位公开条目 ID，可用于详情接口。' },
          { name: 'data.pagination.page', description: '当前页码。' },
          { name: 'data.pagination.limit', description: '实际生效的每页数量。' },
          { name: 'data.pagination.total', description: '当前关键词下可见结果总数。' },
          { name: 'data.pagination.hasMore', description: '是否存在下一页。' }
        ]
      },
      {
        code: 400,
        title: 'Bad request',
        description: '关键词缺失、长度不在 3-100 字符、不是有效 UTF-8，或 page 超过 100。',
        example: `{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request parameters"
  }
}`,
        fields: errorFields
      },
      unauthorizedStatus,
      rateLimitedStatus,
      internalStatus
    ]
  },
  {
    slug: 'game-detail',
    navLabel: '条目详情',
    name: '获取游戏条目详情',
    introduction: '按公开 uniqueId 返回游戏详情、别名、标签、会社与评分聚合。响应不包含内部来源 ID、主站用户、评论或资源下载链接。',
    method: 'GET',
    path: '/v1/games/{uniqueId}',
    auth: '需要有效的 `tgal_live` API token。',
    parameters: [
      ...tokenHeaderParameters,
      {
        name: 'uniqueId',
        location: 'Path',
        required: true,
        type: 'string, 8 alphanumeric chars',
        description: '公开 8 位条目 ID，仅允许英文大小写字母与数字。'
      }
    ],
    requestExample: `curl "https://api.example.com/v1/games/abcd1234" \\
  -H "Authorization: Bearer tgal_live_xxx"`,
    statuses: [
      {
        code: 200,
        title: 'OK',
        description: '返回单个公开条目的完整脱敏元数据。',
        example: `{
  "success": true,
  "data": {
    "uniqueId": "abcd1234",
    "name": "Summer Pockets",
    "aliases": ["サマーポケッツ"],
    "introduction": "公开简介文本。",
    "bannerUrl": "https://example.com/banner.webp",
    "type": ["ADV"],
    "platform": ["PC"],
    "language": ["ja", "zh-Hans"],
    "tags": ["恋爱", "夏日"],
    "publishTime": "2024-01-20T12:00:00Z",
    "releaseDate": "2018-06-29",
    "updatedAt": "2024-06-01T08:30:00Z",
    "resourceUpdateTime": "2024-05-30T10:00:00Z",
    "companies": [
      {
        "name": "Key",
        "aliases": ["VisualArt's Key"]
      }
    ],
    "rating": {
      "average": 8.7,
      "count": 128,
      "recommend": {
        "strongNo": 1,
        "no": 2,
        "neutral": 8,
        "yes": 42,
        "strongYes": 75
      }
    },
    "touchgalUrl": "https://www.touchgal.ink/abcd1234"
  }
}`,
        fields: [
          { name: 'success', description: '固定为 true。' },
          { name: 'data.uniqueId', description: '公开 8 位条目 ID。' },
          { name: 'data.name', description: '条目主名称。' },
          { name: 'data.aliases', description: '公开别名列表，已去重。' },
          { name: 'data.introduction', description: '公开简介文本。' },
          { name: 'data.bannerUrl', description: '公开封面/横幅图片 URL。' },
          { name: 'data.type / platform / language', description: '条目类型、平台和语言数组。' },
          { name: 'data.tags', description: '公开标签名称数组。' },
          { name: 'data.publishTime', description: '条目公开发布时间。' },
          { name: 'data.releaseDate', description: '游戏发售日期文本。' },
          { name: 'data.updatedAt', description: '条目公开元数据更新时间。' },
          { name: 'data.resourceUpdateTime', description: '资源元数据更新时间；不包含资源下载链接。' },
          { name: 'data.companies[].name', description: '会社名称。' },
          { name: 'data.companies[].aliases', description: '会社别名列表。' },
          { name: 'data.rating.average', description: '公开评分均值。' },
          { name: 'data.rating.count', description: '参与评分人数。' },
          { name: 'data.rating.recommend', description: '推荐度直方图：strongNo / no / neutral / yes / strongYes。' },
          { name: 'data.touchgalUrl', description: 'TouchGal 公开条目页面 URL，由 `TOUCHGAL_SITE_URL` 追加 `/{uniqueId}` 生成。' }
        ]
      },
      {
        code: 400,
        title: 'Bad request',
        description: 'uniqueId 不是 8 位，或包含非英文大小写字母/数字字符。',
        example: `{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request parameters"
  }
}`,
        fields: errorFields
      },
      unauthorizedStatus,
      {
        code: 404,
        title: 'Not found',
        description: '未找到该公开 uniqueId，或对应条目已删除/不可公开。',
        example: `{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found"
  }
}`,
        fields: errorFields
      },
      rateLimitedStatus,
      internalStatus
    ]
  }
]

export const getApiEndpointDoc = (slug: string) => apiEndpointDocs.find(doc => doc.slug === slug)
