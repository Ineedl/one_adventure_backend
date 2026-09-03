package refundreport

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicate(t *testing.T) {
	if !isDuplicate(&mysql.MySQLError{Number: 1062, Message: "duplicate"}) {
		t.Fatal("mysql 1062 should be treated as duplicate")
	}
	if isDuplicate(&mysql.MySQLError{Number: 1406, Message: "data too long"}) {
		t.Fatal("non-duplicate mysql error must not be acknowledged")
	}
	if isDuplicate(errors.New("database unavailable")) {
		t.Fatal("generic database error must not be acknowledged")
	}
}
