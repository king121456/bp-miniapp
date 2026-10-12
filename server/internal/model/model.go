package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var CST = time.FixedZone("CST", 8*3600)

type Consent struct {
	AgreedAt any    `json:"agreedAt" bson:"agreedAt"`
	Version  string `json:"version" bson:"version"`
}

type BP struct {
	ID   bson.ObjectID `json:"id" bson:"_id,omitempty"`
	User bson.ObjectID `json:"-" bson:"userId"`
	Time time.Time     `json:"time" bson:"time"`
	Sys  int           `json:"sys" bson:"sys"`
	Dia  int           `json:"dia" bson:"dia"`
	HR   int           `json:"hr" bson:"hr"`
	Tags []string      `json:"tags" bson:"tags"`
	Note string        `json:"note" bson:"note"`
}

type BPIn struct {
	Time *time.Time `json:"time"`
	Sys  int        `json:"sys"`
	Dia  int        `json:"dia"`
	HR   int        `json:"hr"`
	Tags []string   `json:"tags"`
	Note string     `json:"note"`
}

func (in BPIn) Valid() bool {
	return in.Sys >= 40 && in.Sys <= 300 && in.Dia >= 20 && in.Dia <= 200 && in.Dia < in.Sys &&
		(in.HR == 0 || (in.HR >= 20 && in.HR <= 250)) && len(in.Note) <= 500 && len(in.Tags) <= 10
}

type Schedule struct {
	From  string   `json:"from" bson:"from"`
	Times []string `json:"times" bson:"times"`
}

type Med struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	User      bson.ObjectID `json:"-" bson:"userId"`
	Name      string        `json:"name" bson:"name"`
	Dose      string        `json:"dose" bson:"dose"`
	Times     []string      `json:"times" bson:"times"`
	Schedules []Schedule    `json:"-" bson:"schedules,omitempty"`
}

type MedFull struct {
	Med       `bson:",inline"`
	DeletedAt *time.Time `bson:"deletedAt"`
}

func (m Med) TimesOn(date string) []string {
	if len(m.Schedules) == 0 {
		return m.Times
	}
	var t []string
	for _, s := range m.Schedules {
		if s.From <= date {
			t = s.Times
		}
	}
	if t == nil && date >= m.ID.Timestamp().In(CST).Format("2006-01-02") {
		t = m.Schedules[0].Times
	}
	return t
}

func SameTimes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type Measure struct {
	Enabled        bool   `json:"enabled" bson:"enabled"`
	Morning        string `json:"morning" bson:"morning"`
	Evening        string `json:"evening" bson:"evening"`
	MorningEnabled *bool  `json:"morningEnabled" bson:"morningEnabled"`
	EveningEnabled *bool  `json:"eveningEnabled" bson:"eveningEnabled"`
}

type Checkin struct {
	Med         bson.ObjectID `bson:"medId"`
	Date        string        `bson:"date"`
	Slot        string        `bson:"slot"`
	At          time.Time     `bson:"at"`
	IsMakeup    bool          `bson:"isMakeup"`
	RecordedAt  *time.Time    `bson:"recordedAt"`
	MakeupFrom  string        `bson:"makeupFrom"`
	ScheduledAt time.Time     `bson:"scheduledAt"`
}

type Visit struct {
	ID         bson.ObjectID  `json:"id" bson:"_id,omitempty"`
	Patient    bson.ObjectID  `json:"-" bson:"patientId"`
	VisitDate  string         `json:"visitDate" bson:"visitDate"`
	Hospital   string         `json:"hospital" bson:"hospital"`
	Department string         `json:"department" bson:"department"`
	Note       string         `json:"note" bson:"note"`
	RemindTime string         `json:"remindTime" bson:"remindTime"`
	RemindAt   time.Time      `json:"remindAt" bson:"remindAt"`
	Status     string         `json:"status" bson:"status"`
	PushStatus string         `json:"pushStatus" bson:"pushStatus"`
	TodoID     *bson.ObjectID `json:"todoId,omitempty" bson:"todoId,omitempty"`
	CreatedAt  time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt" bson:"updatedAt"`
}

type VisitIn struct {
	VisitDate  string `json:"visitDate"`
	Hospital   string `json:"hospital"`
	Department string `json:"department"`
	Note       string `json:"note"`
	RemindTime string `json:"remindTime"`
	Replace    bool   `json:"replace"`
}

type SlotIn struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

func Today() string { return time.Now().In(CST).Format("2006-01-02") }

func ScheduledAt(date, slot string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+slot, CST)
	return t
}

func Round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }

func Trunc(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func Itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

func MDLabel(date string) string {
	d, err := time.ParseInLocation("2006-01-02", date, CST)
	if err != nil {
		return date
	}
	return Itoa(int(d.Month())) + "月" + Itoa(d.Day()) + "日"
}
