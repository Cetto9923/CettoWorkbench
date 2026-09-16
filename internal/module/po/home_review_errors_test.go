package po

import (
	"context"
	"errors"
	"testing"
)

func TestReviewLookupErrorsReachCallers(t *testing.T) {
	ctx := context.Background()
	req := DemandsReq{Status: "all", Focus: "my_action", Relation: "handling", Page: 1, PageSize: 15}
	cases := map[string]func(*Repo) error{
		"stage summary":  func(r *Repo) error { _, e := r.HomeFocusStageSummary(ctx, "alice", req); return e },
		"focus query":    func(r *Repo) error { _, e := r.homeFocusQuery(ctx, "alice", req); return e },
		"focus count":    func(r *Repo) error { _, e := r.CountHomeFocus(ctx, "alice", req); return e },
		"focus page":     func(r *Repo) error { _, _, e := r.FindHomeFocus(ctx, "alice", req); return e },
		"accept page":    func(r *Repo) error { _, _, e := r.acceptRefsPaged(ctx, "alice", req); return e },
		"all stage page": func(r *Repo) error { _, _, e := r.FindAllStageRefsPaged(ctx, "alice", req); return e },
		"role count": func(r *Repo) error {
			_, e := r.CountRoleDemandsWithFilters(ctx, "alice", mysqlStageFilters["accept"], req)
			return e
		},
		"role page": func(r *Repo) error {
			_, e := r.FindRoleDemandsPagedWithFilters(ctx, "alice", mysqlStageFilters["accept"], req, 0, 15)
			return e
		},
		"role IDs": func(r *Repo) error {
			_, e := r.FindRoleDemandIDsWithFilters(ctx, "alice", mysqlStageFilters["accept"], req)
			return e
		},
		"mixed page": func(r *Repo) error {
			_, _, e := r.FindStageMixedRefsPaged(ctx, "alice", "accept", mysqlStageFilters["accept"], req)
			return e
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			db, mock := openSQLMock(t)
			want := errors.New("review lookup unavailable")
			mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview`").WithArgs("alice").WillReturnError(want)
			if err := call(NewRepo(db, nil)); !errors.Is(err, want) {
				t.Fatalf("want original query error, got %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
