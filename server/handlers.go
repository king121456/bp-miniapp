package main

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func today() string { return time.Now().In(CST).Format("2006-01-02") }

func (a *App) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { ok(w, "ok") })
	m.HandleFunc("POST /api/auth/login", a.login)
	m.HandleFunc("GET /api/me", a.auth(a.me))

	m.HandleFunc("GET /api/bp", a.auth(a.listBP))
	m.HandleFunc("POST /api/bp", a.auth(a.createBP))
	m.HandleFunc("PUT /api/bp/{id}", a.auth(a.updateBP))
	m.HandleFunc("DELETE /api/bp/{id}", a.auth(a.deleteBP))
	m.HandleFunc("GET /api/bp/stats", a.auth(a.statsBP))

	m.HandleFunc("GET /api/meds", a.auth(a.listMeds))
	m.HandleFunc("POST /api/meds", a.auth(a.createMed))
	m.HandleFunc("PUT /api/meds/{id}", a.auth(a.updateMed))
	m.HandleFunc("DELETE /api/meds/{id}", a.auth(a.deleteMed))
	m.HandleFunc("POST /api/meds/{id}/checkin", a.auth(a.checkin))
	m.HandleFunc("GET /api/meds/checkins", a.auth(a.listCheckins))

	m.HandleFunc("GET /api/reminders/measure", a.auth(a.getMeasure))
	m.HandleFunc("PUT /api/reminders/measure", a.auth(a.putMeasure))
	m.HandleFunc("POST /api/subscribe", a.auth(a.addGrant))
	m.HandleFunc("GET /api/todos", a.auth(a.listTodos))
	m.HandleFunc("POST /api/todos/{id}/done", a.auth(a.doneTodo))

	m.HandleFunc("POST /api/family/invite", a.auth(a.invite))
	m.HandleFunc("POST /api/family/bind", a.auth(a.bind))
	m.HandleFunc("GET /api/family", a.auth(a.listFamily))
	m.HandleFunc("DELETE /api/family/{id}", a.auth(a.unbind))
	m.HandleFunc("GET /api/family/patients/{pid}/bp", a.auth(a.familyBP))
	m.HandleFunc("GET /api/family/patients/{pid}/bp/stats", a.auth(a.familyStats))

	m.HandleFunc("GET /api/report", a.auth(a.report))
	m.HandleFunc("GET /api/account/export", a.auth(a.export))
	m.HandleFunc("DELETE /api/account", a.auth(a.deleteAccount))
	return withCORS(m)
}

// withCORS 仅在 APP_ENV=dev 且设置了 CORS_ORIGIN 时生效（用于本机 H5 联调）；
// 生产环境小程序请求不受浏览器 CORS 限制，不开启。CORS_ORIGIN 可为逗号分隔的多个来源或 *。
func withCORS(h http.Handler) http.Handler {
	origins := os.Getenv("CORS_ORIGIN")
	if os.Getenv("APP_ENV") != "dev" || origins == "" {
		return h
	}
	allow := map[string]bool{}
	for _, o := range strings.Split(origins, ",") {
		allow[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && (allow["*"] || allow[o]) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// ---------- auth ----------
type Consent struct {
	AgreedAt any    `json:"agreedAt" bson:"agreedAt"` // 毫秒数字或 RFC3339 字符串都接受
	Version  string `json:"version" bson:"version"`
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string   `json:"code"`
		Nickname string   `json:"nickname"`
		Consent  *Consent `json:"consent"`
	}
	if readJSON(r, &in) != nil {
		fail(w, 400, "invalid json")
		return
	}
	if in.Code == "" {
		fail(w, 400, "code required")
		return
	}
	openid, err := a.wx.Code2Session(r.Context(), in.Code)
	if err != nil {
		fail(w, 502, "wechat login failed")
		return
	}
	set := bson.M{"lastLoginAt": time.Now()}
	if in.Consent != nil && in.Consent.Version != "" {
		set["consent"] = in.Consent
	}
	if in.Nickname != "" {
		set["nickname"] = in.Nickname
	}
	var u struct {
		ID      bson.ObjectID `bson:"_id"`
		Consent *Consent      `bson:"consent"`
	}
	err = a.C("users").FindOneAndUpdate(r.Context(), bson.M{"openid": openid},
		bson.M{"$set": set, "$setOnInsert": bson.M{"createdAt": time.Now(),
			"measure": bson.M{"enabled": false, "morning": "07:30", "evening": "20:00"}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&u)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	if u.Consent == nil {
		fail(w, 403, "consent required")
		return
	}
	ok(w, bson.M{"token": a.sign(u.ID.Hex()), "consent": u.Consent})
}

func (a *App) me(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var u bson.M
	a.C("users").FindOne(r.Context(), bson.M{"_id": uid}, options.FindOne().SetProjection(bson.M{"openid": 0})).Decode(&u)
	ok(w, u)
}

// ---------- blood pressure ----------
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

type bpIn struct {
	Time *time.Time `json:"time"`
	Sys  int        `json:"sys"`
	Dia  int        `json:"dia"`
	HR   int        `json:"hr"`
	Tags []string   `json:"tags"`
	Note string     `json:"note"`
}

// only blocks obvious typos; no medical judgement
func (in bpIn) valid() bool {
	return in.Sys >= 40 && in.Sys <= 300 && in.Dia >= 20 && in.Dia <= 200 && in.Dia < in.Sys &&
		(in.HR == 0 || (in.HR >= 20 && in.HR <= 250)) && len(in.Note) <= 500 && len(in.Tags) <= 10
}

func (a *App) listBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.writeBP(w, r, uid)
}

func (a *App) writeBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	f := bson.M{"userId": uid}
	tr := bson.M{}
	if t, err := time.Parse(time.RFC3339, r.URL.Query().Get("from")); err == nil {
		tr["$gte"] = t
	}
	if t, err := time.Parse(time.RFC3339, r.URL.Query().Get("to")); err == nil {
		tr["$lte"] = t
	}
	if len(tr) > 0 {
		f["time"] = tr
	}
	cur, err := a.C("bp_records").Find(r.Context(), f, options.Find().SetSort(bson.M{"time": -1}).SetLimit(500))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var out []BP
	cur.All(r.Context(), &out)
	for i := range out {
		fixBP(&out[i], a)
	}
	if out == nil {
		out = []BP{}
	}
	ok(w, out)
}

func (a *App) createBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in bpIn
	if readJSON(r, &in) != nil || !in.valid() {
		fail(w, 400, "invalid input")
		return
	}
	t := time.Now()
	if in.Time != nil {
		t = *in.Time
	}
	rec := BP{User: uid, Time: t, Sys: in.Sys, Dia: in.Dia, HR: in.HR, Tags: in.Tags, Note: a.enc(in.Note)}
	res, err := a.C("bp_records").InsertOne(r.Context(), rec)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "measure", "date": today(), "done": false}, bson.M{"$set": bson.M{"done": true}})
	rec.ID = res.InsertedID.(bson.ObjectID)
	fixBP(&rec, a)
	ok(w, bson.M{"record": rec, "needSubscribe": a.needSubscribe(r, uid, "measure")})
}

func (a *App) updateBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	var in bpIn
	if err != nil || readJSON(r, &in) != nil || !in.valid() {
		fail(w, 400, "invalid input")
		return
	}
	set := bson.M{"sys": in.Sys, "dia": in.Dia, "hr": in.HR, "tags": in.Tags, "note": a.enc(in.Note)}
	if in.Time != nil {
		set["time"] = *in.Time
	}
	res, err := a.C("bp_records").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid}, bson.M{"$set": set})
	if err != nil || res.MatchedCount == 0 {
		fail(w, 404, "not found")
		return
	}
	ok(w, nil)
}

func (a *App) deleteBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, err := a.C("bp_records").DeleteOne(r.Context(), bson.M{"_id": id, "userId": uid})
	if err != nil || res.DeletedCount == 0 {
		fail(w, 404, "not found")
		return
	}
	ok(w, nil)
}

type avg struct {
	Sys, Dia, HR float64
	N            int `json:"count"`
}

func (a *App) statsBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	a.writeStats(w, r, uid)
}

// range=week|month ; neutral numbers only, no thresholds
func (a *App) writeStats(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	days := 7
	if r.URL.Query().Get("range") == "month" {
		days = 30
	}
	now := time.Now().In(CST)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, CST).AddDate(0, 0, -(days - 1))
	cur, err := a.C("bp_records").Find(r.Context(), bson.M{"userId": uid, "time": bson.M{"$gte": start}}, options.Find().SetSort(bson.M{"time": 1}))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var recs []BP
	cur.All(r.Context(), &recs)
	type acc struct{ s, d, h, hn, n int }
	add := func(x *acc, b BP) {
		x.s += b.Sys
		x.d += b.Dia
		x.n++
		if b.HR > 0 {
			x.h += b.HR
			x.hn++
		}
	}
	fin := func(x acc) map[string]any {
		if x.n == 0 {
			return map[string]any{"sys": nil, "dia": nil, "hr": nil, "count": 0}
		}
		var hr any
		if x.hn > 0 {
			hr = round1(float64(x.h) / float64(x.hn))
		}
		return map[string]any{"sys": round1(float64(x.s) / float64(x.n)), "dia": round1(float64(x.d) / float64(x.n)), "hr": hr, "count": x.n}
	}
	var all, morning, evening acc
	daily := map[string]*acc{}
	for _, b := range recs {
		t := b.Time.In(CST)
		add(&all, b)
		if t.Hour() < 12 {
			add(&morning, b)
		} else {
			add(&evening, b)
		}
		k := t.Format("2006-01-02")
		if daily[k] == nil {
			daily[k] = &acc{}
		}
		add(daily[k], b)
	}
	points := []map[string]any{}
	for i := 0; i < days; i++ {
		k := start.AddDate(0, 0, i).Format("2006-01-02")
		p := map[string]any{"date": k}
		if d := daily[k]; d != nil {
			for kk, v := range fin(*d) {
				p[kk] = v
			}
		} else {
			p["count"] = 0
		}
		points = append(points, p)
	}
	ok(w, bson.M{"range": days, "average": fin(all), "morning": fin(morning), "evening": fin(evening), "daily": points})
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }

// ---------- meds ----------
type Med struct {
	ID    bson.ObjectID `json:"id" bson:"_id,omitempty"`
	User  bson.ObjectID `json:"-" bson:"userId"`
	Name  string        `json:"name" bson:"name"`
	Dose  string        `json:"dose" bson:"dose"`
	Times []string      `json:"times" bson:"times"`
}

func (a *App) medIn(r *http.Request) (Med, bool) {
	var m Med
	if readJSON(r, &m) != nil || m.Name == "" || len(m.Name) > 100 || len(m.Dose) > 100 || len(m.Times) == 0 || len(m.Times) > 8 {
		return m, false
	}
	for _, t := range m.Times {
		if !hhmm.MatchString(t) {
			return m, false
		}
	}
	sort.Strings(m.Times)
	return m, true
}

func (a *App) listMeds(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("meds").Find(r.Context(), bson.M{"userId": uid})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var meds []Med
	cur.All(r.Context(), &meds)
	date := today()
	cc, _ := a.C("checkins").Find(r.Context(), bson.M{"userId": uid, "date": date})
	var cks []struct {
		Med  bson.ObjectID `bson:"medId"`
		Slot string        `bson:"slot"`
		At   time.Time     `bson:"at"`
	}
	cc.All(r.Context(), &cks)
	done := map[string]time.Time{}
	for _, c := range cks {
		done[c.Med.Hex()+c.Slot] = c.At
	}
	out := []map[string]any{}
	for _, m := range meds {
		slots := []map[string]any{}
		all := true
		for _, t := range m.Times {
			at, d := done[m.ID.Hex()+t]
			s := map[string]any{"time": t, "done": d}
			if d {
				s["doneAt"] = at
			} else {
				all = false
			}
			slots = append(slots, s)
		}
		out = append(out, map[string]any{"id": m.ID, "name": a.dec(m.Name), "dose": a.dec(m.Dose), "times": m.Times, "date": date, "slots": slots, "done": all})
	}
	ok(w, out)
}

func (a *App) createMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	m, good := a.medIn(r)
	if !good {
		fail(w, 400, "invalid input")
		return
	}
	name, dose := m.Name, m.Dose
	m.User, m.Name, m.Dose = uid, a.enc(name), a.enc(dose)
	res, err := a.C("meds").InsertOne(r.Context(), m)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	m.ID, m.Name, m.Dose = res.InsertedID.(bson.ObjectID), name, dose
	ok(w, m)
}

func (a *App) updateMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	m, good := a.medIn(r)
	if err != nil || !good {
		fail(w, 400, "invalid input")
		return
	}
	res, err := a.C("meds").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid},
		bson.M{"$set": bson.M{"name": a.enc(m.Name), "dose": a.enc(m.Dose), "times": m.Times}})
	if err != nil || res.MatchedCount == 0 {
		fail(w, 404, "not found")
		return
	}
	ok(w, nil)
}

func (a *App) deleteMed(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, err := a.C("meds").DeleteOne(r.Context(), bson.M{"_id": id, "userId": uid})
	if err != nil || res.DeletedCount == 0 {
		fail(w, 404, "not found")
		return
	}
	a.C("checkins").DeleteMany(r.Context(), bson.M{"userId": uid, "medId": id})
	ok(w, nil)
}

// body: { "time": "08:00" } ; if omitted and med has one slot, uses it
func (a *App) checkin(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		fail(w, 400, "bad id")
		return
	}
	var in struct {
		Time string `json:"time"`
	}
	readJSON(r, &in)
	var m Med
	if a.C("meds").FindOne(r.Context(), bson.M{"_id": id, "userId": uid}).Decode(&m) != nil {
		fail(w, 404, "not found")
		return
	}
	if in.Time == "" && len(m.Times) == 1 {
		in.Time = m.Times[0]
	}
	valid := false
	for _, t := range m.Times {
		if t == in.Time {
			valid = true
		}
	}
	if !valid {
		fail(w, 400, "time must be one of the med's times")
		return
	}
	_, err = a.C("checkins").UpdateOne(r.Context(), bson.M{"userId": uid, "medId": id, "date": today(), "slot": in.Time},
		bson.M{"$setOnInsert": bson.M{"at": time.Now()}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	a.C("todos").UpdateMany(r.Context(), bson.M{"userId": uid, "kind": "med", "refId": id, "slot": in.Time, "date": today()}, bson.M{"$set": bson.M{"done": true}})
	ok(w, bson.M{"needSubscribe": a.needSubscribe(r, uid, "med")})
}

func (a *App) listCheckins(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	from := time.Now().In(CST).AddDate(0, 0, -30).Format("2006-01-02")
	cur, err := a.C("checkins").Find(r.Context(), bson.M{"userId": uid, "date": bson.M{"$gte": from}}, options.Find().SetSort(bson.M{"at": -1}).SetProjection(bson.M{"_id": 0, "userId": 0}))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var out []bson.M
	cur.All(r.Context(), &out)
	if out == nil {
		out = []bson.M{}
	}
	ok(w, out)
}

// ---------- reminders / subscribe / todos ----------
type Measure struct {
	Enabled        bool   `json:"enabled" bson:"enabled"` // 总开关
	Morning        string `json:"morning" bson:"morning"`
	Evening        string `json:"evening" bson:"evening"`
	MorningEnabled *bool  `json:"morningEnabled" bson:"morningEnabled"` // 不传视为开
	EveningEnabled *bool  `json:"eveningEnabled" bson:"eveningEnabled"`
}

func (a *App) getMeasure(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var u struct {
		Measure Measure `bson:"measure"`
	}
	a.C("users").FindOne(r.Context(), bson.M{"_id": uid}).Decode(&u)
	t := true
	if u.Measure.MorningEnabled == nil {
		u.Measure.MorningEnabled = &t
	}
	if u.Measure.EveningEnabled == nil {
		u.Measure.EveningEnabled = &t
	}
	ok(w, u.Measure)
}

func (a *App) putMeasure(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var m Measure
	if readJSON(r, &m) != nil || (m.Morning != "" && !hhmm.MatchString(m.Morning)) || (m.Evening != "" && !hhmm.MatchString(m.Evening)) {
		fail(w, 400, "invalid input")
		return
	}
	a.C("users").UpdateOne(r.Context(), bson.M{"_id": uid}, bson.M{"$set": bson.M{"measure": m}})
	ok(w, m)
}

var kinds = map[string]bool{"measure": true, "med": true, "family": true}

func (a *App) tpl(kind string) string {
	return map[string]string{"measure": a.cfg.TplMeasure, "med": a.cfg.TplMed, "family": a.cfg.TplFamily}[kind]
}

// called by frontend after wx.requestSubscribeMessage returns "accept"
// body: { "accepted": ["measure","med"] }
func (a *App) addGrant(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in struct {
		Accepted []string `json:"accepted"`
	}
	if readJSON(r, &in) != nil {
		fail(w, 400, "invalid input")
		return
	}
	for _, k := range in.Accepted {
		if !kinds[k] {
			continue
		}
		a.C("subscribe").UpdateOne(r.Context(), bson.M{"userId": uid, "kind": k}, bson.M{"$inc": bson.M{"remaining": 1}}, options.UpdateOne().SetUpsert(true))
	}
	ok(w, a.grants(r, uid))
}

func (a *App) grants(r *http.Request, uid bson.ObjectID) map[string]int {
	cur, _ := a.C("subscribe").Find(r.Context(), bson.M{"userId": uid})
	var gs []struct {
		Kind string `bson:"kind"`
		N    int    `bson:"remaining"`
	}
	cur.All(r.Context(), &gs)
	out := map[string]int{"measure": 0, "med": 0, "family": 0}
	for _, g := range gs {
		out[g.Kind] = g.N
	}
	return out
}

// returns {kind, templateId} when user has <2 pushes left, else nil
func (a *App) needSubscribe(r *http.Request, uid bson.ObjectID, kind string) any {
	if a.grants(r, uid)[kind] >= 2 {
		return nil
	}
	return bson.M{"kind": kind, "templateId": a.tpl(kind)}
}

func (a *App) listTodos(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	cur, err := a.C("todos").Find(r.Context(), bson.M{"userId": uid, "done": false}, options.Find().SetSort(bson.M{"createdAt": -1}).SetLimit(50).SetProjection(bson.M{"userId": 0}))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var out []bson.M
	cur.All(r.Context(), &out)
	if out == nil {
		out = []bson.M{}
	}
	ok(w, out)
}

func (a *App) doneTodo(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	a.C("todos").UpdateOne(r.Context(), bson.M{"_id": id, "userId": uid}, bson.M{"$set": bson.M{"done": true}})
	ok(w, nil)
}

// ---------- family ----------
func code6() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return big.NewInt(0).Add(n, big.NewInt(100000)).String()
}

func (a *App) invite(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	exp := time.Now().Add(24 * time.Hour)
	for i := 0; i < 5; i++ {
		c := code6()
		if _, err := a.C("invites").InsertOne(r.Context(), bson.M{"code": c, "patientId": uid, "expiresAt": exp}); err == nil {
			ok(w, bson.M{"code": c, "expiresAt": exp})
			return
		}
	}
	fail(w, 500, "try again")
}

func (a *App) bind(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var in struct {
		Code string `json:"code"`
	}
	readJSON(r, &in)
	var inv struct {
		Patient bson.ObjectID `bson:"patientId"`
	}
	err := a.C("invites").FindOneAndDelete(r.Context(), bson.M{"code": in.Code, "expiresAt": bson.M{"$gt": time.Now()}}).Decode(&inv)
	if err != nil {
		fail(w, 404, "invalid or expired code")
		return
	}
	if inv.Patient == uid {
		fail(w, 400, "cannot bind yourself")
		return
	}
	_, err = a.C("family").InsertOne(r.Context(), bson.M{"patientId": inv.Patient, "memberId": uid, "createdAt": time.Now()})
	if mongo.IsDuplicateKeyError(err) {
		fail(w, 409, "already bound")
		return
	}
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, bson.M{"patientId": inv.Patient})
}

// returns who can see me (members) and whom I can see (patients)
func (a *App) listFamily(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	load := func(f bson.M, other string) []bson.M {
		cur, _ := a.C("family").Find(r.Context(), f)
		var links []bson.M
		cur.All(r.Context(), &links)
		out := []bson.M{}
		for _, l := range links {
			var u struct {
				Nickname string `bson:"nickname"`
			}
			a.C("users").FindOne(r.Context(), bson.M{"_id": l[other]}).Decode(&u)
			out = append(out, bson.M{"id": l["_id"], "userId": l[other], other: l[other], "nickname": u.Nickname, "createdAt": l["createdAt"]})
		}
		return out
	}
	ok(w, bson.M{"members": load(bson.M{"patientId": uid}, "memberId"), "patients": load(bson.M{"memberId": uid}, "patientId")})
}

// either side can unbind
func (a *App) unbind(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	id, _ := bson.ObjectIDFromHex(r.PathValue("id"))
	res, _ := a.C("family").DeleteOne(r.Context(), bson.M{"_id": id, "$or": bson.A{bson.M{"patientId": uid}, bson.M{"memberId": uid}}})
	if res == nil || res.DeletedCount == 0 {
		fail(w, 404, "not found")
		return
	}
	ok(w, nil)
}

func (a *App) familyPatient(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) (bson.ObjectID, bool) {
	pid, err := bson.ObjectIDFromHex(r.PathValue("pid"))
	if err == nil {
		if n, _ := a.C("family").CountDocuments(r.Context(), bson.M{"patientId": pid, "memberId": uid}); n > 0 {
			return pid, true
		}
	}
	fail(w, 403, "forbidden")
	return pid, false
}

func (a *App) familyBP(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	if pid, good := a.familyPatient(w, r, uid); good {
		a.writeBP(w, r, pid)
	}
}
func (a *App) familyStats(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	if pid, good := a.familyPatient(w, r, uid); good {
		a.writeStats(w, r, pid)
	}
}

// ---------- account ----------
func (a *App) export(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	var bps []BP
	cur, _ := a.C("bp_records").Find(r.Context(), bson.M{"userId": uid}, options.Find().SetSort(bson.M{"time": 1}))
	cur.All(r.Context(), &bps)
	bps = append([]BP{}, bps...)
	for i := range bps {
		fixBP(&bps[i], a)
	}
	var meds []Med
	cur, _ = a.C("meds").Find(r.Context(), bson.M{"userId": uid})
	cur.All(r.Context(), &meds)
	for i := range meds {
		meds[i].Name, meds[i].Dose = a.dec(meds[i].Name), a.dec(meds[i].Dose)
	}
	meds = append([]Med{}, meds...)
	cks := []bson.M{}
	cur, _ = a.C("checkins").Find(r.Context(), bson.M{"userId": uid}, options.Find().SetProjection(bson.M{"_id": 0, "userId": 0}))
	cur.All(r.Context(), &cks)
	w.Header().Set("Content-Disposition", `attachment; filename="bp-export.json"`)
	ok(w, bson.M{"exportedAt": time.Now(), "bloodPressure": bps, "medications": meds, "checkins": cks})
}

// hard delete everything belonging to the user
func (a *App) deleteAccount(w http.ResponseWriter, r *http.Request, uid bson.ObjectID) {
	ctx := r.Context()
	for _, c := range []string{"bp_records", "meds", "checkins", "subscribe", "todos"} {
		if _, err := a.C(c).DeleteMany(ctx, bson.M{"userId": uid}); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	a.C("invites").DeleteMany(ctx, bson.M{"patientId": uid})
	a.C("family").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"patientId": uid}, bson.M{"memberId": uid}}})
	a.C("users").DeleteOne(ctx, bson.M{"_id": uid})
	ok(w, nil)
}

// fixBP 解密备注、统一北京时间、空 tags 返回 []
func fixBP(b *BP, a *App) {
	b.Note = a.dec(b.Note)
	b.Time = b.Time.In(CST)
	if b.Tags == nil {
		b.Tags = []string{}
	}
}
