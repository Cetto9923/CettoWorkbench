package query

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newQueryRepoMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return NewRepo(db), mock
}

func TestListDemandsMatchesDisplayedOwnerAndSystem(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	like := "%项目%"
	ownerLike := "%周鸿利%"
	countSQL := `(?s)SELECT count\(\*\) FROM .*zt_demand d.*LOWER\(COALESCE\(cu\.realname, d\.BRA\)\) LIKE.*LOWER\(COALESCE\(p\.name, d\.mainSystem\)\) LIKE.*EXISTS \(.*zt_demandclarify fdc.*LOWER\(COALESCE\(fcp\.name, fdc\.product\)\) LIKE`
	mock.ExpectQuery(countSQL).WithArgs("0", ownerLike, ownerLike, like, like).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*LOWER\(COALESCE\(cu\.realname, d\.BRA\)\) LIKE.*ORDER BY d\.id DESC`).WithArgs("0", ownerLike, ownerLike, like, like, 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).AddRow(1, "需求", "3", "active", "", "周鸿利(004861)", "项目管理系统2.0", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Owner: "周鸿利", System: "项目", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Rows) != 1 || resp.Rows[0].Owner != "周鸿利(004861)" || resp.Rows[0].System != "项目管理系统2.0" {
		t.Fatalf("unexpected filtered response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListStoriesFiltersAndReturnsSystem(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	ownerLike := "%周鸿利%"
	systemLike := "%项目%"
	countSQL := regexp.QuoteMeta("SELECT count(*) FROM zt_story s")
	mock.ExpectQuery(`(?s)`+countSQL+`.*LOWER\(COALESCE\(su\.realname, s\.assignedTo\)\) LIKE.*LOWER\(COALESCE\(sp\.name, CAST\(s\.product AS CHAR\)\)\) LIKE`).WithArgs("0", 0, "story", ownerLike, ownerLike, systemLike).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_story s.*LEFT JOIN zt_product sp.*LOWER\(COALESCE\(su\.realname, s\.assignedTo\)\) LIKE.*ORDER BY s\.id DESC`).WithArgs("0", 0, "story", ownerLike, ownerLike, systemLike, 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).AddRow(2, "研发需求", 2, "active", "", "周鸿利", "项目管理系统2.0", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Tab: tabRD, Owner: "周鸿利", System: "项目", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Rows) != 1 || resp.Rows[0].System != "项目管理系统2.0" {
		t.Fatalf("unexpected filtered response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
