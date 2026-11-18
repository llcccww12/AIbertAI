# 灵感导航 Logo

运行时只使用本目录资源，不请求远程图标服务。标志保留原色：默认使用白色衬底，Savee 的透明白标、Design360 的高亮绿标与 Stash 的灰色字标使用深色衬底，确保明暗主题均可见。

本次替换的资源：

- `film-grab.com.png`：官网声明的站点图标，来源 `https://film-grab.com/wp-content/uploads/2019/02/icon.jpg`，缩放为 64px PNG。
- `shot.cafe.png`：官网 `https://shot.cafe/stream/` 的 `.navbar-brand` 完整 `SHOT.CAFE` 字标截图，保留字形、拼写和颜色；该站点未声明有效 favicon，原 ICO 是全透明空图，不以首字母或通用图标代替。
- `thedieline.com.png`：域名 `www.thedieline.com` 的公开 favicon 缓存，来源 `https://www.google.com/s2/favicons?domain=www.thedieline.com&sz=128`；与网站实际 D 形标志核对后保存为 64px PNG，替换原透明 ICO。
- `stashmedia.tv.png`：官网声明的 `https://www.stashmedia.tv/wp-content/uploads/cropped-Stash-Favicon-2017-icon-192x192.png`，缩放为 64px PNG，去除白色背景及抗锯齿边缘中的白色底色后使用深色衬底；保留原始灰色主色 RGB(188,190,192)、S 字形及透明抗锯齿轮廓，替换旧的浅灰白低对比 ICO。
- `the-brandidentity.com.png`：官网声明的 `https://the-brandidentity.com/icons/favicon-32x32.png`。官方 favicon 本身为纯蓝色方块，并非空图，保留官方设计。

检查不能只判断文件是否存在：运行 `node apps/web/test/browser-inspiration.browser.mjs`，会逐一解码全部 79 个 Logo 并检查非空白、明暗主题中的像素可见度及所有轮换窗口。首字回退只用于意外加载失败，不算正常 Logo 验收通过。

标志权利归各网站及其权利人所有，仅用于识别导航目的地。
