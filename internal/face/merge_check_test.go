package face

import (
	"context"
	"errors"
	"testing"
)

type mergeCheckStub struct{ calls int }

func (s *mergeCheckStub) GetPersonMergeCheck(_ context.Context, source, target string) (PersonMergeCheck, error) {
	s.calls++
	return PersonMergeCheck{SourcePersonID: source, TargetPersonID: target, SourceRevision: 2, TargetRevision: 3}, nil
}
func TestPersonMergeCheckServiceValidatesDistinctPeople(t *testing.T) {
	if _, err := NewPersonMergeCheckService(nil); err == nil {
		t.Fatal("nil repository accepted")
	}
	repo := &mergeCheckStub{}
	service, _ := NewPersonMergeCheckService(repo)
	for _, pair := range [][2]string{{"", "person_valid"}, {"person_valid", "person_valid"}, {"../bad", "person_valid"}} {
		if _, err := service.Check(context.Background(), pair[0], pair[1]); !errors.Is(err, ErrInvalidPerson) {
			t.Fatal(err)
		}
	}
	if repo.calls != 0 {
		t.Fatal("invalid requests reached repository")
	}
	value, err := service.Check(context.Background(), "person_source", "person_target")
	if err != nil || !value.CanMerge() || repo.calls != 1 {
		t.Fatalf("%+v %v", value, err)
	}
	for _, value := range []PersonMergeCheck{{CannotLinkCount: 1}, {DuplicateAssignmentCount: 1}, {SourceLimitExceeded: true}} {
		if value.CanMerge() {
			t.Fatal("blocked merge allowed")
		}
	}
}
