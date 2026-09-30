# 画布 TV · 主播子域域名与 CORS

父域默认 **`huabutv.com`**。画布 host 默认 **`www.huabutv.com`**。可用环境变量覆盖：`CANVAS_PUBLIC_PARENT_DOMAIN`、`CANVAS_PUBLIC_AGENT_HOST`、`CANVAS_PUBLIC_CANVAS_HOST`。

Cookie 已知父域是品牌常量 **`{huabutv.com, j11.net}`**，与 `CANVAS_PUBLIC_PARENT_DOMAIN` **解耦**（空 backup env 也拿不掉 j11）。产品 UI / 导航 / 页脚 / 主播后台只展示 huabutv 主机，**不出现** `j11.net` URL。

## 主机怎么分

| 主机 | 用途 |
|---|---|
| `www.huabutv.com` | canonical 官网 / 登录后画布 |
| `huabutv.com` | apex，**HTTPS 301 → www** |
| `canvas.huabutv.com` | 保留字，**HTTPS 301 → www**（不是并列官网） |
| `agent.huabutv.com` | 代理后台 |
| `{slug}.huabutv.com` | 该主播营销皮 + 锁定邀请码 |
| `cdn.huabutv.com` | 对象 CDN：**CNAME → 阿里云 CDN**，不要 A 到 ECS |
| `oss.huabutv.com` | OSS 自定义域：**CNAME → bucket**，禁止 A 到 ECS |
| `mail.huabutv.com` | DirectMail：**MX + TXT**（不要给 `mail` 做 CNAME） |
| `canvas.j11.net` / `agent.j11.net` / `{slug}.j11.net` | **备用入口**：继续反代同一套 web，**不 301**；仅运维文档 |

不能当主播 slug：`www` / `app` / `api` / `agent` / `admin` / `static` / `cdn` / `mail` / `canvas` / **`oss` / `smtp` / `track` / `mx` / `ns` / `email`**。

## DNS / 证书 / nginx（运维，非本仓配置）

cdn / oss 的 **CNAME 先于** 通配 `*.huabutv.com` A。MX/TXT **压不过** 通配 A：`mail` 的 HTTPS 仍会打到 ECS，必须靠更精确的 nginx `server_name` **`return 444`**（`smtp` / `track` / `mx` / `ns` / `email` / `cdn` / `oss` 同理，禁止 `proxy_pass` 进 SPA）。

ECS 通配证书走现网同一条路径：

```bash
acme.sh --issue -d huabutv.com -d '*.huabutv.com' --dns dns_ali
```

**不是**阿里云免费 DV 通配（免费 DV 只支持单域名，给 `cdn` / `oss` 用）。j11 通配证继续给备用域。

阶段 2 HSTS 只写 `max-age=31536000`，**禁止** `includeSubDomains` / preload。等 `https://cdn.huabutv.com/` 与 `https://oss.huabutv.com/` 都出示匹配证书后再加 `includeSubDomains`。

## Cookie / CORS

- Cookie：Host 落在已知父域上时写 `.`+该父域。现网混合 env（`CANVAS_PUBLIC_PARENT_DOMAIN=j11.net` + `CANVAS_COOKIE_DOMAIN=.j11.net`）访问 `www.huabutv.com` 仍必须 `Set-Cookie; Domain=.huabutv.com`。两套登录态不共享。
- CORS 现网至少（**apex 必须显式列入**，www 带不出 apex；禁止 `*`）：

```
CANVAS_CORS_ORIGINS=https://www.huabutv.com,https://huabutv.com,https://canvas.j11.net
```

该名单会放行 `https://agent.huabutv.com`、`https://{slug}.huabutv.com` 以及 `https://a.j11.net` 这一级子域。

## 硬顺序

**PR-A（品牌常量双父域 Cookie）→ nginx 阶段 2 → 观察到 `Set-Cookie; Domain=.huabutv.com` 才把人放到 www → 阶段 4 env → PR-B → 阶段 5 `CDNBaseURL`。**

在 curl 看到该 Set-Cookie 之前，不要把人类流量导向 www。未完成 ICP 备案则 **禁止** 把国内 CDN 自定义域切到生产。

## 流量

1. 用户打开 `https://{slug}.huabutv.com` → 该主播登录/注册皮（邀请码锁死）
2. 登录成功 → `https://www.huabutv.com` 同一套画布
3. 主播打开 `https://agent.huabutv.com` → 只读后台
4. `https://huabutv.com` 与 `https://canvas.huabutv.com` → 301 到 www
