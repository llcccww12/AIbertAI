package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsTerminal struct {
	in, out *os.File
	console windows.Handle
	job     windows.Handle
	shell   string
	done    chan int
	mu      sync.Mutex
}

func startTerminalProcess(cwd string, size terminalSize) (terminalProcess, error) {
	if err := windows.NewLazySystemDLL("kernel32.dll").NewProc("CreatePseudoConsole").Find(); err != nil {
		return nil, err
	}
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return nil, err
	}
	shell := filepath.Join(system, "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(shell); err != nil {
		shell = filepath.Join(system, "cmd.exe")
	}
	shellPtr, err := windows.UTF16PtrFromString(shell)
	if err != nil {
		return nil, err
	}
	cwdPtr, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return nil, err
	}
	commandLine, err := windows.UTF16PtrFromString(windows.EscapeArg(shell))
	if err != nil {
		return nil, err
	}
	env := utf16.Encode([]rune(strings.Join(terminalEnvironment(), "\x00") + "\x00\x00"))

	var inputRead, inputWrite, outputRead, outputWrite windows.Handle
	if err := windows.CreatePipe(&inputRead, &inputWrite, nil, 0); err != nil {
		return nil, err
	}
	defer windows.CloseHandle(inputRead)
	in := os.NewFile(uintptr(inputWrite), "terminal-input")
	if err := windows.CreatePipe(&outputRead, &outputWrite, nil, 0); err != nil {
		_ = in.Close()
		return nil, err
	}
	defer windows.CloseHandle(outputWrite)
	out := os.NewFile(uintptr(outputRead), "terminal-output")
	ok := false
	defer func() {
		if !ok {
			_ = in.Close()
			_ = out.Close()
		}
	}()
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !ok {
			_ = windows.CloseHandle(job)
		}
	}()
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return nil, err
	}
	var console windows.Handle
	if err := windows.CreatePseudoConsole(windows.Coord{X: int16(size.Cols), Y: int16(size.Rows)}, inputRead, outputWrite, 0, &console); err != nil {
		return nil, err
	}
	defer func() {
		if !ok {
			// Older Windows versions can wait for the output pipe when closing.
			_ = out.Close()
			windows.ClosePseudoConsole(console)
		}
	}()
	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, err
	}
	defer attributes.Delete()
	if err := setConsoleAttribute(attributes, console); err != nil {
		return nil, err
	}
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	var process windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_SUSPENDED)
	if err := windows.CreateProcess(shellPtr, commandLine, nil, nil, false, flags, &env[0], cwdPtr, &startup.StartupInfo, &process); err != nil {
		return nil, err
	}
	defer windows.CloseHandle(process.Thread)
	defer func() {
		if !ok {
			_ = windows.TerminateProcess(process.Process, 1)
			_ = windows.CloseHandle(process.Process)
		}
	}()
	if err := windows.AssignProcessToJobObject(job, process.Process); err != nil {
		return nil, err
	}
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		return nil, err
	}
	t := &windowsTerminal{in: in, out: out, console: console, job: job, shell: filepath.Base(shell), done: make(chan int, 1)}
	ok = true
	go t.join(process.Process)
	return t, nil
}

func setConsoleAttribute(attributes *windows.ProcThreadAttributeListContainer, console windows.Handle) error {
	// Unlike Go pointers to attribute data, HPCON is passed as the handle value.
	update := windows.NewLazySystemDLL("kernel32.dll").NewProc("UpdateProcThreadAttribute")
	result, _, err := update.Call(uintptr(unsafe.Pointer(attributes.List())), 0, windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		uintptr(console), unsafe.Sizeof(console), 0, 0)
	runtime.KeepAlive(attributes)
	if result == 0 {
		return err
	}
	return nil
}

func (t *windowsTerminal) Read(p []byte) (int, error)  { return t.out.Read(p) }
func (t *windowsTerminal) Write(p []byte) (int, error) { return t.in.Write(p) }
func (t *windowsTerminal) Shell() string               { return t.shell }
func (t *windowsTerminal) Wait() int {
	code := <-t.done
	_ = t.out.Close()
	return code
}

func (t *windowsTerminal) Resize(size terminalSize) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.console == 0 {
		return os.ErrClosed
	}
	return windows.ResizePseudoConsole(t.console, windows.Coord{X: int16(size.Cols), Y: int16(size.Rows)})
}

func (t *windowsTerminal) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		_ = windows.TerminateJobObject(t.job, 1)
	}
}

func (t *windowsTerminal) join(process windows.Handle) {
	defer windows.CloseHandle(process)
	var code uint32
	wait, err := windows.WaitForSingleObject(process, windows.INFINITE)
	if err != nil || wait != windows.WAIT_OBJECT_0 || windows.GetExitCodeProcess(process, &code) != nil {
		code = 1
	}
	t.mu.Lock()
	console, job := t.console, t.job
	t.console, t.job = 0, 0
	t.mu.Unlock()
	_ = windows.CloseHandle(job)
	_ = t.in.Close()
	// The terminal reader continues draining output while ConPTY shuts down.
	windows.ClosePseudoConsole(console)
	t.done <- int(code)
}
