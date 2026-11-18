import AppKit

let project = Bundle.main.bundleURL.deletingLastPathComponent()
let script = project.appendingPathComponent("启动员工端.command")

func showLaunchError(_ message: String) {
    let app = NSApplication.shared
    app.setActivationPolicy(.accessory)
    app.activate(ignoringOtherApps: true)
    let alert = NSAlert()
    alert.alertStyle = .warning
    alert.messageText = "无法启动 AIbertAI"
    alert.informativeText = message
    alert.runModal()
}

guard FileManager.default.isExecutableFile(atPath: script.path) else {
    showLaunchError("请把 AIbertAI.app 放在 AIbertAI 项目目录下，与启动员工端.command 放在一起。")
    exit(EXIT_FAILURE)
}

let desktop = Process()
desktop.executableURL = script
desktop.currentDirectoryURL = project
var environment = ProcessInfo.processInfo.environment
environment["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
desktop.environment = environment
let diagnostics = Pipe()
desktop.standardError = diagnostics

do {
    try desktop.run()
    let output = diagnostics.fileHandleForReading.readDataToEndOfFile()
    desktop.waitUntilExit()
    if desktop.terminationStatus != 0 {
        let details = String(decoding: output, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines)
        showLaunchError(details.isEmpty ? "请在项目的 .local-runtime/desktop.log 查看启动日志。" : details)
        exit(EXIT_FAILURE)
    }
} catch {
    showLaunchError(error.localizedDescription)
    exit(EXIT_FAILURE)
}
