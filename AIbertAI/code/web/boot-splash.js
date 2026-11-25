// 窗口在整页脚本载入后才显示；隐藏期间动画时钟照走，显示时会直接跳到结尾。
// 隐藏的窗口只会渲染首帧，所以等到第二帧真正画出来再从头播放启动动画。
(function () {
  var splash = document.getElementById("boot-splash");
  if (!splash) return;
  splash.classList.add("bs-wait");
  requestAnimationFrame(function () {
    requestAnimationFrame(function () {
      splash.classList.remove("bs-wait");
      window.__bootSplashShownAt = performance.now();
      window.dispatchEvent(new Event("aibertai:boot-splash-shown"));
    });
  });
})();
