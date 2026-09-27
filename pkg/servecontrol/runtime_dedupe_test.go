package servecontrol

import "testing"

func TestRuntimeDeduplicatesEquivalentLogDiagnostics(t *testing.T) {
	runtime := NewRuntime()
	jobID := runtime.QueueJob(JobSpec{Name: "build", Type: "build"})
	otherJobID := runtime.QueueJob(JobSpec{Name: "other", Type: "build"})

	runtime.AddLog(LogEntry{Level: "warning", Message: " warning: missing   title ", JobID: jobID})
	runtime.AddDiagnostic(Diagnostic{Code: "serve.log_warning", Severity: "warning", Message: " warning: missing   title ", JobID: jobID})
	runtime.AddDiagnostic(Diagnostic{Code: "frontmatter.invalid_title", Severity: "warning", Message: "missing title", File: "one.md", Line: 4, JobID: jobID})
	runtime.AddDiagnostic(Diagnostic{Code: "frontmatter.invalid_title", Severity: "warning", Message: "missing title", File: "two.md", Line: 7, JobID: jobID})
	runtime.AddDiagnostic(Diagnostic{Code: "serve.log_warning", Severity: "warning", Message: "another warning", JobID: jobID})
	runtime.AddDiagnostic(Diagnostic{Code: "serve.log_warning", Severity: "warning", Message: "missing title", JobID: otherJobID})

	snapshot := runtime.Snapshot()
	if len(snapshot.Diagnostics) != 4 {
		t.Fatalf("inbox diagnostics = %+v", snapshot.Diagnostics)
	}
	if len(snapshot.Jobs[0].Diagnostics) != 3 {
		t.Fatalf("first job diagnostics = %+v", snapshot.Jobs[0].Diagnostics)
	}
	if len(snapshot.Jobs[1].Diagnostics) != 1 {
		t.Fatalf("other job diagnostic was incorrectly deduplicated: %+v", snapshot.Jobs[1].Diagnostics)
	}
	if len(snapshot.Logs) != 1 || len(snapshot.Jobs[0].Logs) != 1 {
		t.Fatalf("source log should remain visible, logs=%+v job logs=%+v", snapshot.Logs, snapshot.Jobs[0].Logs)
	}
}
