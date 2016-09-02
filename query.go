package clickhouse

import (
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Query struct {
	Stmt   string
	NbCols uint64
	args   []interface{}
}

func (q *Query) Args() []interface{} { return q.args }
func (q *Query) DeleteRow(rowID uint64) {
	start, end := uint64(rowID)*q.NbCols, uint64(rowID)*q.NbCols+q.NbCols
	q.args = append(q.args[:start], q.args[uint64(rowID)*q.NbCols+q.NbCols:]...)
	log.Warnf("discarded data:\n%#v", q.args[start:end])
}

func (q Query) Iter(conn *Conn) *Iter {
	if conn == nil {
		return &Iter{err: errors.New("Connection pointer is nil")}
	}
	resp, err := conn.transport.Exec(conn, q, false)
	if err != nil {
		return &Iter{err: err}
	}

	err = errorFromResponse(resp)
	if err != nil {
		return &Iter{err: err}
	}

	return &Iter{text: resp}
}

func (q Query) Exec(conn *Conn) (err error) {
	if conn == nil {
		return errors.New("Connection pointer is nil")
	}
	resp, err := conn.transport.Exec(conn, q, false)
	if err == nil {
		err = errorFromResponse(resp)
	}

	return err
}

type Iter struct {
	err  error
	text string
}

func (r *Iter) Error() error {
	return r.err
}

func (r *Iter) Scan(vars ...interface{}) bool {
	row := r.fetchNext()
	if len(row) == 0 {
		return false
	}
	a := strings.Split(row, "\t")
	if len(a) < len(vars) {
		return false
	}
	for i, v := range vars {
		err := unmarshal(v, a[i])
		if err != nil {
			r.err = err
			return false
		}
	}
	return true
}

func (r *Iter) fetchNext() string {
	var res string
	pos := strings.Index(r.text, "\n")
	if pos == -1 {
		res = r.text
		r.text = ""
	} else {
		res = r.text[:pos]
		r.text = r.text[pos+1:]
	}
	return res
}
