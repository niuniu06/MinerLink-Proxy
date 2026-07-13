package proxy

import (
	"fmt"
	"testing"
)

func TestSession_JobTracker_FeeMode(t *testing.T) {
	s := &Session{
		jobTracker: make(map[string]FeeMode),
		jobList:    make([]string, 0),
	}

	// 1. Add Main Job
	s.addJob("job_main_1", FeeModeNone)
	mode, exists := s.checkJobFeeMode("job_main_1")
	if !exists || mode != FeeModeNone {
		t.Errorf("Expected job_main_1 to be FeeModeNone, got %v (exists: %v)", mode, exists)
	}

	// 2. Add Dev Fee Job
	s.addJob("job_dev_1", FeeModeDev)
	mode, exists = s.checkJobFeeMode("job_dev_1")
	if !exists || mode != FeeModeDev {
		t.Errorf("Expected job_dev_1 to be FeeModeDev, got %v", mode)
	}

	// 3. Add Operator Fee Job
	s.addJob("job_op_1", FeeModeOperator)
	mode, exists = s.checkJobFeeMode("job_op_1")
	if !exists || mode != FeeModeOperator {
		t.Errorf("Expected job_op_1 to be FeeModeOperator, got %v", mode)
	}

	// 4. Case insensitivity check
	s.addJob("JOB_UPPER", FeeModeDev)
	mode, exists = s.checkJobFeeMode("job_upper")
	if !exists || mode != FeeModeDev {
		t.Errorf("Expected checkJobFeeMode to be case insensitive, got exists: %v", exists)
	}

	// 5. Non-existent Job
	mode, exists = s.checkJobFeeMode("job_unknown")
	if exists {
		t.Errorf("Expected job_unknown to not exist")
	}
}

func TestSession_JobTracker_LRUEviction(t *testing.T) {
	s := &Session{
		jobTracker: make(map[string]FeeMode),
		jobList:    make([]string, 0),
	}

	// MaxTrackedJobs is 10000
	// We will add 10005 jobs and verify the first 5 are evicted

	for i := 0; i < 10005; i++ {
		jobID := fmt.Sprintf("job_%d", i)
		s.addJob(jobID, FeeModeNone)
	}

	if len(s.jobTracker) != MaxTrackedJobs {
		t.Errorf("Expected jobTracker map size to be capped at %d, got %d", MaxTrackedJobs, len(s.jobTracker))
	}

	if len(s.jobList) != MaxTrackedJobs {
		t.Errorf("Expected jobList array size to be capped at %d, got %d", MaxTrackedJobs, len(s.jobList))
	}

	// job_0 to job_4 should be evicted
	for i := 0; i < 5; i++ {
		jobID := fmt.Sprintf("job_%d", i)
		_, exists := s.checkJobFeeMode(jobID)
		if exists {
			t.Errorf("Expected %s to be evicted, but it exists", jobID)
		}
	}

	// job_5 to job_10004 should exist
	_, exists := s.checkJobFeeMode("job_5")
	if !exists {
		t.Errorf("Expected job_5 to exist")
	}

	_, exists = s.checkJobFeeMode("job_10004")
	if !exists {
		t.Errorf("Expected job_10004 to exist")
	}
}
