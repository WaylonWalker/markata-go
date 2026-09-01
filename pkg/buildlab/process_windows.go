//go:build windows

package buildlab

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcessTree struct {
	job windows.Handle
	pid uint32
}

func newProcessTree(*exec.Cmd) (processTree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	return &windowsProcessTree{job: job}, nil
}

func (p *windowsProcessTree) Start(cmd *exec.Cmd) error {
	if p.job == 0 {
		return errors.New("Build Lab process job is closed")
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Keep the process suspended until it has been assigned to the job. This
	// closes the window in which the root could create an uncontained child.
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	if err := cmd.Start(); err != nil {
		return errors.Join(err, p.Close())
	}
	p.pid = uint32(cmd.Process.Pid)
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, p.pid)
	if err != nil {
		return p.startFailure(cmd, err, false)
	}
	assignErr := windows.AssignProcessToJobObject(p.job, process)
	closeErr := windows.CloseHandle(process)
	if assignErr != nil {
		return p.startFailure(cmd, errors.Join(assignErr, closeErr), false)
	}
	if closeErr != nil {
		return p.startFailure(cmd, closeErr, true)
	}
	if err := resumeProcessThread(p.pid); err != nil {
		return p.startFailure(cmd, err, true)
	}
	return nil
}

func (p *windowsProcessTree) startFailure(cmd *exec.Cmd, startErr error, assigned bool) error {
	var cleanupErr error
	if assigned {
		cleanupErr = windows.TerminateJobObject(p.job, 1)
	}
	if !assigned || cleanupErr != nil {
		if cmd.Process != nil {
			cleanupErr = errors.Join(cleanupErr, cmd.Process.Kill())
		}
	}
	if cmd.Process != nil {
		cleanupErr = errors.Join(cleanupErr, cmd.Wait())
	}
	cleanupErr = errors.Join(cleanupErr, p.Close())
	return errors.Join(startErr, cleanupErr)
}

func resumeProcessThread(pid uint32) (err error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, windows.CloseHandle(snapshot))
	}()
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	if err := windows.Thread32First(snapshot, &entry); err != nil {
		return err
	}
	var lastErr error
	for {
		if entry.OwnerProcessID == pid {
			thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if openErr == nil {
				previous, resumeErr := windows.ResumeThread(thread)
				closeErr := windows.CloseHandle(thread)
				if resumeErr == nil && previous == 0 {
					resumeErr = errors.New("primary process thread was not suspended")
				}
				return errors.Join(resumeErr, closeErr)
			}
			lastErr = openErr
		}
		if nextErr := windows.Thread32Next(snapshot, &entry); nextErr != nil {
			if errors.Is(nextErr, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nextErr
		}
	}
	if lastErr != nil {
		return fmt.Errorf("open suspended Build Lab process thread: %w", lastErr)
	}
	return fmt.Errorf("find suspended Build Lab process thread for pid %d", pid)
}

func (p *windowsProcessTree) Kill() error {
	if p.job == 0 {
		return nil
	}
	return windows.TerminateJobObject(p.job, 1)
}

func (p *windowsProcessTree) Close() error {
	if p.job == 0 {
		return nil
	}
	err := windows.CloseHandle(p.job)
	p.job = 0
	p.pid = 0
	return err
}
