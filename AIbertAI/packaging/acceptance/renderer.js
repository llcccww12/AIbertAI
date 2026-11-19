"use strict";

const element = (id) => document.getElementById(id);
const run = element("run");
const authorize = element("authorize");
const status = element("status");
const answer = element("answer");

window.acceptance.deployment().then(({ backendOrigin, modelName }) => {
  element("backend").textContent = backendOrigin;
  element("model").textContent = modelName;
});

element("form").addEventListener("submit", async (event) => {
  event.preventDefault();
  run.disabled = true;
  answer.textContent = "";
  status.textContent = "正在联系私有后端…";
  try {
    await window.acceptance.run(element("prompt").value);
  } catch (error) {
    status.textContent = error.message;
    run.disabled = false;
  }
});

authorize.addEventListener("click", async () => {
  try {
    await window.acceptance.authorize();
  } catch {
    status.textContent = "无法打开浏览器授权页面，请重试。";
  }
});

window.acceptance.onEvent(({ type, value }) => {
  switch (type) {
    case "authorize":
      authorize.hidden = false;
      status.textContent = "请在浏览器完成私有后端登录及授权。";
      break;
    case "login":
      authorize.hidden = true;
      status.textContent = "登录成功，正在验证设备和模型。";
      break;
    case "delta":
      answer.append(value);
      break;
    case "reported":
      status.textContent = "后端已确认会话上报 · " + value;
      break;
    case "error":
      status.textContent = value;
      break;
    case "finished":
      run.disabled = false;
      authorize.hidden = true;
      break;
  }
});
