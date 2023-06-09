package model

import "database/sql"

type Training struct {
	ID         int            `json:"id"`
	Sequence   int            `json:"sequence"`
	Topic      string         `json:"topic"`
	Name       string         `json:"name"`
	URL        string         `json:"url"`
	IsFree     bool           `json:"is_free"`
	ProjectURL sql.NullString `json:"project_url"`
	CourseID   int            `json:"course_id"`
}
