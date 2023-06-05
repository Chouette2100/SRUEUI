package srdblc

import (
	"strconv"
	"strings"

	"bytes"
	"fmt"
	"log"
	"os"

	//	"math"
	"sort"
	"time"

	"net/http"

	"github.com/PuerkitoBio/goquery"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/srapi"

	"SRUEUI/srapilc"
)

/*
Ver.01AA00	新規作成
Ver.01AB00	現在開催中のイベントのみをたいしょうにする（SelectLastEventList() ==> SelectCurEventList()）
*/

const Version = "01AB01"

type DBConfig struct {
	WebServer string `yaml:"WebServer"`
	HTTPport  string `yaml:"HTTPport"`
	SSLcrt    string `yaml:"SSLcrt"`
	SSLkey    string `yaml:"SSLkey"`
	Dbhost    string `yaml:"Dbhost"`
	Dbname    string `yaml:"Dbname"`
	Dbuser    string `yaml:"Dbuser"`
	Dbpw      string `yaml:"Dbpw"`
	Sracct    string `yaml:"Sracct"`
	Srpswd    string `yaml:"Srpswd"`
}

var Dbconfig *DBConfig

type Event_Inf struct {
	Event_ID    string
	I_Event_ID  int
	Event_name  string
	Event_no    int
	MaxPoint    int
	Start_time  time.Time
	Sstart_time string
	Start_date  float64
	End_time    time.Time
	Send_time   string
	Period      string
	Dperiod     float64
	Intervalmin int
	Modmin      int
	Modsec      int
	Fromorder   int
	Toorder     int
	Resethh     int
	Resetmm     int
	Nobasis     int
	Maxdsp      int
	NoEntry     int
	NoRoom      int    //	ルーム数
	EventStatus string //	"Over", "BeingHeld", "NotHeldYet"
	Pntbasis    int
	Ordbasis    int
	League_ids  string
	Cmap        int
	Target      int
	Maxpoint    int
	//	Status		string		//	"Confirmed":	イベント終了日翌日に確定した獲得ポイントが反映されている。
}

type Color struct {
	Name  string
	Value string
}

// https://www.fukushihoken.metro.tokyo.lg.jp/kiban/machizukuri/kanren/color.files/colorudguideline.pdf
var Colorlist2 []Color = []Color{
	{"red", "#FF2800"},
	{"yellow", "#FAF500"},
	{"green", "#35A16B"},
	{"blue", "#0041FF"},
	{"skyblue", "#66CCFF"},
	{"lightpink", "#FFD1D1"},
	{"orange", "#FF9900"},
	{"purple", "#9A0079"},
	{"brown", "#663300"},
	{"lightgreen", "#87D7B0"},
	{"white", "#FFFFFF"},
	{"gray", "#77878F"},
}

var Colorlist1 []Color = []Color{
	{"cyan", "cyan"},
	{"magenta", "magenta"},
	{"yellow", "yellow"},
	{"royalblue", "royalblue"},
	{"coral", "coral"},
	{"khaki", "khaki"},
	{"deepskyblue", "deepskyblue"},
	{"crimson", "crimson"},
	{"orange", "orange"},
	{"lightsteelblue", "lightsteelblue"},
	{"pink", "pink"},
	{"sienna", "sienna"},
	{"springgreen", "springgreen"},
	{"blueviolet", "blueviolet"},
	{"salmon", "salmon"},
	{"lime", "lime"},
	{"red", "red"},
	{"darkorange", "darkorange"},
	{"skyblue", "skyblue"},
	{"lightpink", "lightpink"},
}

type ColorInf struct {
	Color      string
	Colorvalue string
	Selected   string
}

type ColorInfList []ColorInf

type RoomInfo struct {
	Name      string //	ルーム名のリスト
	Longname  string
	Shortname string
	Account   string //	アカウントのリスト、アカウントは配信のURLの最後の部分の英数字です。
	ID        string //	IDのリスト、IDはプロフィールのURLの最後の部分で5～6桁の数字です。
	Userno    int
	//	APIで取得できるデータ(1)
	Genre      string
	Rank       string
	Irank      int
	Nrank      string
	Prank      string
	Followers  int
	Sfollowers string
	Fans       int
	Fans_lst   int
	Level      int
	Slevel     string
	//	APIで取得できるデータ(2)
	Order        int
	Point        int //	イベント終了後12時間〜36時間はイベントページから取得できることもある
	Spoint       string
	Istarget     string
	Graph        string
	Iscntrbpoint string
	Color        string
	Colorvalue   string
	Colorinflist ColorInfList
	Formid       string
	Eventid      string
	Status       string
	Statuscolor  string
}

type RoomInfoList []RoomInfo

// sort.Sort()のための関数三つ
func (r RoomInfoList) Len() int {
	return len(r)
}

func (r RoomInfoList) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RoomInfoList) Choose(from, to int) (s RoomInfoList) {
	s = r[from:to]
	return
}

var SortByFollowers bool

// 降順に並べる
func (r RoomInfoList) Less(i, j int) bool {
	//	return e[i].point < e[j].point
	if SortByFollowers {
		return r[i].Followers > r[j].Followers
	} else {
		return r[i].Point > r[j].Point
	}
}

type PerSlot struct {
	Timestart time.Time
	Dstart    string
	Tstart    string
	Tend      string
	Point     string
	Ipoint    int
	Tpoint    string
}

type PerSlotInf struct {
	Eventname   string
	Eventid     string
	Period      string
	Roomname    string
	Roomid      int
	Perslotlist []PerSlot
}

var Event_inf Event_Inf

var Db *sql.DB
var Err error

func OpenDb() (status int) {

	status = 0

	//	https://leben.mobi/go/mysql-connect/practice/
	//	OS := runtime.GOOS

	//	https://ssabcire.hatenablog.com/entry/2019/02/13/000722
	//	https://konboi.hatenablog.com/entry/2016/04/12/100903
	/*
		switch OS {
		case "windows":
			Db, Err = sql.Open("mysql", wuser+":"+wpw+"@/"+wdb+"?parseTime=true&loc=Asia%2FTokyo")
		case "linux":
			Db, Err = sql.Open("mysql", luser+":"+lpw+"@/"+ldb+"?parseTime=true&loc=Asia%2FTokyo")
		case "freebsd":
			//	https://leben.mobi/go/mysql-connect/practice/
			Db, Err = sql.Open("mysql", buser+":"+bpw+"@tcp("+bhost+":3306)/"+bdb+"?parseTime=true&loc=Asia%2FTokyo")
		default:
			log.Printf("%s is not supported.\n", OS)
			status = -2
		}
	*/

	if (*Dbconfig).Dbhost == "" {
		Db, Err = sql.Open("mysql", (*Dbconfig).Dbuser+":"+(*Dbconfig).Dbpw+"@/"+(*Dbconfig).Dbname+"?parseTime=true&loc=Asia%2FTokyo")
	} else {
		Db, Err = sql.Open("mysql", (*Dbconfig).Dbuser+":"+(*Dbconfig).Dbpw+"@tcp("+(*Dbconfig).Dbhost+":3306)/"+(*Dbconfig).Dbname+"?parseTime=true&loc=Asia%2FTokyo")
	}

	if Err != nil {
		status = -1
	}
	return
}

//	現在開催中のイベントのリストを作る
func SelectCurEventList() (eventlist []Event_Inf, status int) {

	var stmt *sql.Stmt
	var rows *sql.Rows

	sql := "select eventid, event_name, period, starttime, endtime, fromorder, toorder from event "
	sql += " where endtime > now() and starttime < now() "
	stmt, Err = Db.Prepare(sql)
	if Err != nil {
		log.Printf("err=[%s]\n", Err.Error())
		status = -1
		return
	}
	defer stmt.Close()

	rows, Err = stmt.Query()
	if Err != nil {
		log.Printf("err=[%s]\n", Err.Error())
		status = -1
		return
	}
	defer rows.Close()

	var event Event_Inf
	for rows.Next() {
		Err = rows.Scan(&event.Event_ID, &event.Event_name, &event.Period, &event.Start_time, &event.End_time, &event.Fromorder, &event.Toorder)
		if Err != nil {
			log.Printf("err=[%s]\n", Err.Error())
			status = -1
			return
		}
		eventlist = append(eventlist, event)
	}
	if Err = rows.Err(); Err != nil {
		log.Printf("err=[%s]\n", Err.Error())
		status = -1
		return
	}

	return

}

func SelectEventNoAndName(eventid string) (
	eventname string,
	period string,
	status int,
) {

	status = 0

	err := Db.QueryRow("select event_name, period from event where eventid ='"+eventid+"'").Scan(&eventname, &period)

	if err == nil {
		return
	} else {
		log.Printf("err=[%s]\n", err.Error())
		if err.Error() != "sql: no rows in result set" {
			status = -2
			return
		}
	}

	status = -1
	return
}

func SelectEventInf(eventid string) (eventinf Event_Inf, status int) {

	status = 0

	sql := "select eventid,ieventid,event_name, period, starttime, endtime, noentry, intervalmin, modmin, modsec, "
	sql += " Fromorder, Toorder, Resethh, Resetmm, Nobasis, Maxdsp, cmap, target, maxpoint "
	sql += " from event where eventid = ?"
	err := Db.QueryRow(sql, eventid).Scan(
		&eventinf.Event_ID,
		&eventinf.I_Event_ID,
		&eventinf.Event_name,
		&eventinf.Period,
		&eventinf.Start_time,
		&eventinf.End_time,
		&eventinf.NoEntry,
		&eventinf.Intervalmin,
		&eventinf.Modmin,
		&eventinf.Modsec,
		&eventinf.Fromorder,
		&eventinf.Toorder,
		&eventinf.Resethh,
		&eventinf.Resetmm,
		&eventinf.Nobasis,
		&eventinf.Maxdsp,
		&eventinf.Cmap,
		&eventinf.Target,
		&eventinf.Maxpoint,
	)

	if err != nil {
		log.Printf("%s\n", sql)
		log.Printf("err=[%s]\n", err.Error())
		//	if err.Error() != "sql: no rows in result set" {
		status = -1
		return
		//	}
	}

	//	log.Printf("eventno=%d\n", Event_inf.Event_no)

	start_date := eventinf.Start_time.Truncate(time.Hour).Add(-time.Duration(eventinf.Start_time.Hour()) * time.Hour)
	end_date := eventinf.End_time.Truncate(time.Hour).Add(-time.Duration(eventinf.End_time.Hour())*time.Hour).AddDate(0, 0, 1)

	//	log.Printf("start_t=%v\nstart_d=%v\nend_t=%v\nend_t=%v\n", Event_inf.Start_time, start_date, Event_inf.End_time, end_date)

	eventinf.Start_date = float64(start_date.Unix()) / 60.0 / 60.0 / 24.0
	eventinf.Dperiod = float64(end_date.Unix())/60.0/60.0/24.0 - Event_inf.Start_date

	//	log.Printf("Start_data=%f Dperiod=%f\n", eventinf.Start_date, eventinf.Dperiod)

	return
}

func SelectEventRoomInfList(
	eventid string,
	roominfolist *RoomInfoList,
) (
	eventname string,
	status int,
) {

	status = 0

	//	eventno := 0
	//	eventno, eventname, _ = SelectEventNoAndName(eventid)
	Event_inf, _ = SelectEventInf(eventid)

	//	eventno := Event_inf.Event_no
	eventname = Event_inf.Event_name

	sql := "select distinct u.userno, userid, user_name, longname, shortname, genre, `rank`, nrank, prank, level, followers, fans, fans_lst, e.istarget, e.graph, e.color, e.iscntrbpoints, e.point "
	sql += " from user u join eventuser e "
	sql += " where u.userno = e.userno and e.eventid= ?"
	if Event_inf.Start_time.After(time.Now()) {
		sql += " order by followers desc"
	} else {
		sql += " order by e.point desc"
	}

	stmt, err := Db.Prepare(sql)
	if err != nil {
		log.Printf("SelectEventRoomInfList() Prepare() err=%s\n", err.Error())
		status = -5
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(eventid)
	if err != nil {
		log.Printf("SelectRoomIn() Query() (6) err=%s\n", err.Error())
		status = -6
		return
	}
	defer rows.Close()

	ColorlistA := Colorlist2
	ColorlistB := Colorlist1
	if Event_inf.Cmap == 1 {
		ColorlistA = Colorlist1
		ColorlistB = Colorlist2
	}

	colormap := make(map[string]int)

	for i := 0; i < len(ColorlistA); i++ {
		colormap[ColorlistA[i].Name] = i
	}

	var roominf RoomInfo

	i := 0
	for rows.Next() {
		err := rows.Scan(&roominf.Userno,
			&roominf.Account,
			&roominf.Name,
			&roominf.Longname,
			&roominf.Shortname,
			&roominf.Genre,
			&roominf.Rank,
			&roominf.Nrank,
			&roominf.Prank,
			&roominf.Level,
			&roominf.Followers,
			&roominf.Fans,
			&roominf.Fans_lst,
			&roominf.Istarget,
			&roominf.Graph,
			&roominf.Color,
			&roominf.Iscntrbpoint,
			&roominf.Point,
		)

		ci := 0
		for ; ci < len(ColorlistA); ci++ {
			if ColorlistA[ci].Name == roominf.Color {
				roominf.Colorvalue = ColorlistA[ci].Value
				break
			}
		}
		if ci == len(ColorlistA) {
			ci := 0
			for ; ci < len(ColorlistB); ci++ {
				if ColorlistB[ci].Name == roominf.Color {
					roominf.Colorvalue = ColorlistB[ci].Value
					break
				}
			}
			if ci == len(ColorlistB) {
				roominf.Colorvalue = roominf.Color
			}
		}

		if roominf.Istarget == "Y" {
			roominf.Istarget = "Checked"
		} else {
			roominf.Istarget = ""
		}
		if roominf.Graph == "Y" {
			roominf.Graph = "Checked"
		} else {
			roominf.Graph = ""
		}
		if roominf.Iscntrbpoint == "Y" {
			roominf.Iscntrbpoint = "Checked"
		} else {
			roominf.Iscntrbpoint = ""
		}
		roominf.Slevel = humanize.Comma(int64(roominf.Level))
		roominf.Sfollowers = humanize.Comma(int64(roominf.Followers))
		if roominf.Point < 0 {
			roominf.Spoint = ""
		} else {
			roominf.Spoint = humanize.Comma(int64(roominf.Point))
		}
		roominf.Formid = "Form" + fmt.Sprintf("%d", i)
		roominf.Eventid = eventid
		roominf.Name = strings.ReplaceAll(roominf.Name, "'", "’")
		if err != nil {
			log.Printf("SelectEventRoomInfList() Scan() err=%s\n", err.Error())
			status = -7
			return
		}
		//	var colorinf ColorInf
		colorinflist := make([]ColorInf, len(ColorlistA))

		for i := 0; i < len(ColorlistA); i++ {
			colorinflist[i].Color = ColorlistA[i].Name
			colorinflist[i].Colorvalue = ColorlistA[i].Value
		}

		roominf.Colorinflist = colorinflist
		if cidx, ok := colormap[roominf.Color]; ok {
			roominf.Colorinflist[cidx].Selected = "Selected"
		}
		*roominfolist = append(*roominfolist, roominf)

		i++
	}

	if err = rows.Err(); err != nil {
		log.Printf("SelectEventRoomInfList() rows err=%s\n", err.Error())
		status = -8
		return
	}

	if Event_inf.Start_time.After(time.Now()) {
		SortByFollowers = true
	} else {
		SortByFollowers = false
	}
	sort.Sort(*roominfolist)

	/*
		for i := 0; i < len(*roominfolist); i++ {

			sql = "select max(point) from points where "
			sql += " user_id = " + fmt.Sprintf("%d", (*roominfolist)[i].Userno)
			//	sql += " and event_id = " + fmt.Sprintf("%d", eventno)
			sql += " and event_id = " + eventid

			err = Db.QueryRow(sql).Scan(&(*roominfolist)[i].Point)
			(*roominfolist)[i].Spoint = humanize.Comma(int64((*roominfolist)[i].Point))

			if err == nil {
				continue
			} else {
				log.Printf("err=[%s]\n", err.Error())
				if err.Error() != "sql: no rows in result set" {
					eventno = -2
					continue
				} else {
					(*roominfolist)[i].Point = -1
					(*roominfolist)[i].Spoint = ""
				}
			}
		}
	*/

	return
}

func SelectPointList(userno int, eventid string) (norow int, tp *[]time.Time, pp *[]int) {

	norow = 0

	//	log.Printf("SelectPointList() userno=%d eventid=%s\n", userno, eventid)
	stmt1, err := Db.Prepare("SELECT count(*) FROM points where user_id = ? and eventid = ?")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer stmt1.Close()

	//	var norow int
	err = stmt1.QueryRow(userno, eventid).Scan(&norow)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	//	fmt.Println(norow)

	//	----------------------------------------------------

	//	stmt1, err = Db.Prepare("SELECT max(t.t) FROM timeacq t join points p where t.idx=p.idx and user_id = ? and event_id = ?")
	stmt1, err = Db.Prepare("SELECT max(ts) FROM points where user_id = ? and eventid = ?")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer stmt1.Close()

	var tfinal time.Time
	err = stmt1.QueryRow(userno, eventid).Scan(&tfinal)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	islastdata := false
	if tfinal.After(Event_inf.End_time.Add(time.Duration(-Event_inf.Intervalmin) * time.Minute)) {
		islastdata = true
	}
	//	fmt.Println(norow)

	//	----------------------------------------------------

	t := make([]time.Time, norow)
	point := make([]int, norow)
	if islastdata {
		t = make([]time.Time, norow+1)
		point = make([]int, norow+1)
	}

	tp = &t
	pp = &point

	if norow == 0 {
		return
	}

	//	----------------------------------------------------

	//	stmt2, err := Db.Prepare("select t.t, p.point from points p join timeacq t on t.idx = p.idx where user_id = ? and event_id = ? order by t.t")
	stmt2, err := Db.Prepare("select ts, point from points where user_id = ? and eventid = ? order by ts")
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer stmt2.Close()

	rows, err := stmt2.Query(userno, eventid)
	if err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		err := rows.Scan(&t[i], &point[i])
		if err != nil {
			//	log.Fatal(err)
			log.Printf("err=[%s]\n", err.Error())
			//	status = -1
			return
		}
		i++

	}
	if err = rows.Err(); err != nil {
		//	log.Fatal(err)
		log.Printf("err=[%s]\n", err.Error())
		//	status = -1
		return
	}

	if islastdata {
		t[norow] = t[norow-1].Add(15 * time.Minute)
		point[norow] = point[norow-1]
	}

	tp = &t
	pp = &point

	return
}

func UpdatePointsSetQstatus(
	eventid string,
	userno int,
	tstart string,
	tend string,
	point string,
) (status int) {
	status = 0

	log.Printf("  *** UpdatePointsSetQstatus() *** eventid=%s userno=%d\n", eventid, userno)

	nrow := 0
	//	err := Db.QueryRow("select count(*) from points where eventid = ? and user_id = ? and pstatus = 'Conf.'", eventid, userno).Scan(&nrow)
	sql := "select count(*) from points where eventid = ? and user_id = ? and ( pstatus = 'Conf.' or pstatus = 'Prov.' )"
	err := Db.QueryRow(sql, eventid, userno).Scan(&nrow)

	if err != nil {
		log.Printf("select count(*) from user ... err=[%s]\n", err.Error())
		status = -1
		return
	}

	if nrow != 1 {
		return
	}

	log.Printf("  *** UpdatePointsSetQstatus() Update!\n")

	sql = "update points set qstatus =?,"
	sql += "qtime=? "
	//	sql += "where user_id=? and eventid = ? and pstatus = 'Conf.'"
	sql += "where user_id=? and eventid = ? and ( pstatus = 'Conf.' or pstatus = 'Prov.' )"
	stmt, err := Db.Prepare(sql)
	if err != nil {
		log.Printf("UpdatePointsSetQstatus() Update/Prepare err=%s\n", err.Error())
		status = -1
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(point, tstart+"--"+tend, userno, eventid)

	if err != nil {
		log.Printf("error(UpdatePointsSetQstatus() Update/Exec) err=%s\n", err.Error())
		status = -2
	}

	return
}

func MakePointPerSlot(eventid string) (perslotinflist []PerSlotInf, status int) {

	var perslotinf PerSlotInf
	var event_inf Event_Inf

	status = 0

	event_inf.Event_ID = eventid
	//	eventno, eventname, period := SelectEventNoAndName(eventid)
	eventname, period, _ := SelectEventNoAndName(eventid)

	var roominfolist RoomInfoList

	_, sts := SelectEventRoomInfList(eventid, &roominfolist)

	if sts != 0 {
		log.Printf("status of SelectEventRoomInfList() =%d\n", sts)
		status = sts
		return
	}

	var perslot PerSlot

	for i := 0; i < len(roominfolist); i++ {

		if roominfolist[i].Graph != "Checked" {
			continue
		}

		userid := roominfolist[i].Userno

		perslotinf.Eventname = eventname
		perslotinf.Eventid = eventid
		perslotinf.Period = period

		perslotinf.Roomname = roominfolist[i].Name
		perslotinf.Roomid = userid
		perslotinf.Perslotlist = make([]PerSlot, 0)

		norow, tp, pp := SelectPointList(userid, eventid)

		if norow == 0 {
			continue
		}

		sameaslast := true
		plast := (*pp)[0]
		pprv := (*pp)[0]
		tdstart := ""
		tstart := time.Now().Truncate(time.Second)

		for i, t := range *tp {
			//	if (*pp)[i] != plast && sameaslast {
			if (*pp)[i] != plast {
				tstart = t
				/*
					if i != 0 {
						log.Printf("(1) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
					} else {
						log.Printf("(1) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
					}
				*/
				if sameaslast {
					//	これまで変化しなかった獲得ポイントが変化し始めた
					pdstart := t.Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("2006/01/02")
					if pdstart != tdstart {
						perslot.Dstart = pdstart
						tdstart = pdstart
					} else {
						perslot.Dstart = ""
					}
					perslot.Timestart = t.Add(time.Duration(-Event_inf.Modmin) * time.Minute)
					//	perslot.Tstart = t.Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("15:04")
					if t.Sub((*tp)[i-1]) < 31*time.Minute {
						perslot.Tstart = perslot.Timestart.Format("15:04")
					} else {
						perslot.Tstart = "n/a"
					}
					//	perslot.Tstart = perslot.Timestart.Format("15:04")

					sameaslast = false
					//	} else if (*pp)[i] == plast && !sameaslast && (*tp)[i].Sub((*tp)[i-1]) > 11*time.Minute {
				}
			} else if (*pp)[i] == plast {
				//	if !sameaslast && (*tp)[i].Sub((*tp)[i-1]) > 16*time.Minute {
				if !sameaslast && t.Sub(tstart) > 11*time.Minute {
					//	if !sameaslast {
					/*
						if i != 0 {
							log.Printf("(2) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
						} else {
							log.Printf("(2) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
						}
					*/
					if perslot.Tstart != "n/a" {
						perslot.Tend = (*tp)[i-1].Add(time.Duration(-Event_inf.Modmin) * time.Minute).Format("15:04")
					} else {
						perslot.Tend = "n/a"
					}
					perslot.Ipoint = plast - pprv
					perslot.Point = humanize.Comma(int64(plast - pprv))
					perslot.Tpoint = humanize.Comma(int64(plast))
					sameaslast = true
					perslotinf.Perslotlist = append(perslotinf.Perslotlist, perslot)
					pprv = plast
				}
				//	sameaslast = true
			}
			/* else
			{
					if i != 0 {
						log.Printf("(3) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, (*tp)[i-1]=%s\n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"), (*tp)[i-1].Format("01/02 15:04"))
					} else {
						log.Printf("(3) (*pp)[i]=%d, plast=%d, sameaslast=%v, (*tp)[i]=%s, \n", (*pp)[i], plast, sameaslast, (*tp)[i].Format("01/02 15:04"))
					}
			}
			*/
			plast = (*pp)[i]
		}

		if len(perslotinf.Perslotlist) != 0 {
			perslotinflist = append(perslotinflist, perslotinf)
		}

		UpdatePointsSetQstatus(eventid, userid, perslot.Tstart, perslot.Tend, perslot.Point)

	}

	return
}

func GetAndInsertEventRoomInfo(
	client *http.Client,
	eventid string,
	breg int,
	ereg int,
	eventinfo *Event_Inf,
	roominfolist *RoomInfoList,
) (
	starttimeafternow bool,
	status int,
) {

	log.Println("GetAndInsertEventRoomInfo() Called.")
	log.Println(*eventinfo)

	status = 0
	starttimeafternow = false

	//	イベントに参加しているルームの一覧を取得します。
	//	ルーム名、ID、URLを取得しますが、イベント終了直後の場合の最終獲得ポイントが表示されている場合はそれも取得します。

	if strings.Contains(eventid, "?") {
		status = GetEventInfAndRoomListBR(client, eventid, breg, ereg, eventinfo, roominfolist)
		eia := strings.Split(eventid, "?")
		bka := strings.Split(eia[1], "=")
		eventinfo.Event_name = eventinfo.Event_name + "(" + bka[1] + ")"
	} else {
		status = GetEventInfAndRoomList(eventid, breg, ereg, eventinfo, roominfolist)
	}

	if status != 0 {
		log.Printf("GetEventInfAndRoomList() returned %d\n", status)
		return
	}

	//	各ルームのジャンル、ランク、レベル、フォロワー数を取得します。
	for i := 0; i < (*eventinfo).NoRoom; i++ {
		(*roominfolist)[i].Genre, (*roominfolist)[i].Rank,
			(*roominfolist)[i].Nrank,
			(*roominfolist)[i].Prank,
			(*roominfolist)[i].Level,
			(*roominfolist)[i].Followers,
			(*roominfolist)[i].Fans,
			(*roominfolist)[i].Fans_lst,
			_, _, _, _ = srapilc.GetRoomInfoByAPI((*roominfolist)[i].ID)

	}

	//	各ルームの獲得ポイントを取得します。
	for i := 0; i < (*eventinfo).NoRoom; i++ {
		point, _, _, eventid := srapilc.GetPointsByAPI((*roominfolist)[i].ID)
		if eventid == (*eventinfo).Event_ID {
			(*roominfolist)[i].Point = point
			UpdateEventuserSetPoint(eventid, (*roominfolist)[i].ID, point)
			if point < 0 {
				(*roominfolist)[i].Spoint = ""
			} else {
				(*roominfolist)[i].Spoint = humanize.Comma(int64(point))
			}
		} else {
			log.Printf(" %s %s %d\n", (*eventinfo).Event_ID, eventid, point)
		}

		if (*roominfolist)[i].ID == fmt.Sprintf("%d", (*eventinfo).Nobasis) {
			(*eventinfo).Pntbasis = point
			(*eventinfo).Ordbasis = i
		}

		//	log.Printf(" followers=<%d> level=<%d> nrank=<%s> genre=<%s> point=%d\n",
		//	(*roominfolist)[i].Followers,
		//	(*roominfolist)[i].Level,
		//	(*roominfolist)[i].Nrank,
		//	(*roominfolist)[i].Genre,
		//	(*roominfolist)[i].Point)
	}

	if (*eventinfo).Start_time.After(time.Now()) {
		SortByFollowers = true
		sort.Sort(*roominfolist)
		if ereg > len(*roominfolist) {
			ereg = len(*roominfolist)
		}
		r := (*roominfolist).Choose(breg-1, ereg)
		roominfolist = &r
		starttimeafternow = true
	}

	log.Printf(" GetEventRoomInfo() len(*roominfolist)=%d\n", len(*roominfolist))

	log.Println("GetAndInsertEventRoomInfo() before InsertEventIinf()")
	log.Println(*eventinfo)
	status = InsertEventInf(eventinfo)

	if status == 1 {
		log.Println("InsertEventInf() returned 1.")
		UpdateEventInf(eventinfo)
		status = 0
	}
	log.Println("GetAndInsertEventRoomInfo() after InsertEventIinf() or UpdateEventInf")
	log.Println(*eventinfo)

	_, _, status = SelectEventNoAndName(eventid)

	if status == 0 {
		//	InsertRoomInf(eventno, eventid, roominfolist)
		InsertRoomInf(eventid, roominfolist)
	}

	return
}

func GetEventInfAndRoomListBR(
	client *http.Client,
	eventid string,
	breg int,
	ereg int,
	eventinfo *Event_Inf,
	roominfolist *RoomInfoList,
) (
	status int,
) {

	status = 0

	var doc *goquery.Document
	var err error

	inputmode := "url"
	eventidorfilename := eventid

	status = 0

	//	URLからドキュメントを作成します
	_url := "https://www.showroom-live.com/event/" + eventidorfilename
	/*
		doc, err = goquery.NewDocument(_url)
	*/
	resp, error := http.Get(_url)
	if error != nil {
		log.Printf("GetEventInfAndRoomList() http.Get() err=%s\n", error.Error())
		status = 1
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	/*
	bufstr := buf.String()
	log.Printf("%s\n", bufstr)
	*/

	//	doc, error = goquery.NewDocumentFromReader(resp.Body)
	doc, error = goquery.NewDocumentFromReader(buf)
	if error != nil {
		log.Printf("GetEventInfAndRoomList() goquery.NewDocumentFromReader() err=<%s>.\n", error.Error())
		status = 1
		return
	}

	(*eventinfo).Event_ID = eventidorfilename
	//	log.Printf(" eventid=%s\n", (*eventinfo).Event_ID)

	cevent_id, exists := doc.Find("#eventDetail").Attr("data-event-id")
	if !exists {
		log.Printf("data-event-id not found. Event_ID=%s\n", (*eventinfo).Event_ID)
		status = -1
		return
	}
	eventinfo.I_Event_ID, _ = strconv.Atoi(cevent_id)
	event_id := eventinfo.I_Event_ID

	selector := doc.Find(".detail")
	(*eventinfo).Event_name = selector.Find(".tx-title").Text()
	if (*eventinfo).Event_name == "" {
		log.Printf("Event not found. Event_ID=%s\n", (*eventinfo).Event_ID)
		status = -1
		return
	}
	(*eventinfo).Period = selector.Find(".info").Text()
	eventinfo.Period = strings.Replace(eventinfo.Period, "\u202f", " ", -1)
	period := strings.Split((*eventinfo).Period, " - ")
	if inputmode == "url" {
		(*eventinfo).Start_time, _ = time.Parse("Jan 2, 2006 3:04 PM MST", period[0]+" JST")
		(*eventinfo).End_time, _ = time.Parse("Jan 2, 2006 3:04 PM MST", period[1]+" JST")
	} else {
		(*eventinfo).Start_time, _ = time.Parse("2006/01/02 15:04 MST", period[0]+" JST")
		(*eventinfo).End_time, _ = time.Parse("2006/01/02 15:04 MST", period[1]+" JST")
	}

	(*eventinfo).EventStatus = "BeingHeld"
	if (*eventinfo).Start_time.After(time.Now()) {
		(*eventinfo).EventStatus = "NotHeldYet"
	} else if (*eventinfo).End_time.Before(time.Now()) {
		(*eventinfo).EventStatus = "Over"
	}

	//	イベントに参加しているルームの数を求めます。
	//	参加ルーム数と表示されているルームの数は違うので、ここで取得したルームの数を以下の処理で使うわけではありません。
	SNoEntry := doc.Find("p.ta-r").Text()
	fmt.Sscanf(SNoEntry, "%d", &((*eventinfo).NoEntry))
	log.Printf("[%s]\n[%s] [%s] (*event).EventStatus=%s NoEntry=%d\n",
		(*eventinfo).Event_name,
		(*eventinfo).Start_time.Format("2006/01/02 15:04 MST"),
		(*eventinfo).End_time.Format("2006/01/02 15:04 MST"),
		(*eventinfo).EventStatus, (*eventinfo).NoEntry)
	log.Printf("breg=%d ereg=%d\n", breg, ereg)

	//	eventno, _, _ := SelectEventNoAndName(eventidorfilename)
	//	log.Printf(" eventno=%d\n", eventno)
	//	(*eventinfo).Event_no = eventno

	eia := strings.Split(eventid, "?")
	bia := strings.Split(eia[1], "=")
	blockid, _ := strconv.Atoi(bia[1])

	/*
		event_id := 30030
		event_id := 31947
	*/

	ebr, err := srapi.GetEventBlockRanking(client, event_id, blockid, breg, ereg)
	if err != nil {
		log.Printf("GetEventBlockRanking() err=%s\n", err.Error())
		status = 1
		return
	}

	ReplaceString := "/r/"

	for _, br := range ebr.Block_ranking_list {

		var roominfo RoomInfo

		roominfo.ID = br.Room_id
		roominfo.Userno, _ = strconv.Atoi(roominfo.ID)

		roominfo.Account = strings.Replace(br.Room_url_key, ReplaceString, "", -1)
		roominfo.Account = strings.Replace(roominfo.Account, "/", "", -1)

		roominfo.Name = br.Room_name

		*roominfolist = append(*roominfolist, roominfo)

	}

	(*eventinfo).NoRoom = len(*roominfolist)

	log.Printf(" GetEventInfAndRoomList() len(*roominfolist)=%d\n", len(*roominfolist))

	return
}

func UpdateEventuserSetPoint(eventid, userid string, point int) (status int) {
	status = 0

	//	eventno, _, _ := SelectEventNoAndName(eventid)
	userno, _ := strconv.Atoi(userid)

	sql := "update eventuser set point=? where eventid = ? and userno = ?"
	stmt, err := Db.Prepare(sql)
	if err != nil {
		log.Printf("UpdateEventuserSetPoint() error (Update/Prepare) err=%s\n", err.Error())
		status = -1
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(point, eventid, userno)

	if err != nil {
		log.Printf("error(UpdateEventuserSetPoint() Update/Exec) err=%s\n", err.Error())
		status = -2
	}

	return
}

func InsertEventInf(eventinf *Event_Inf) (
	status int,
) {

	if _, _, status = SelectEventNoAndName((*eventinf).Event_ID); status != 0 {
		sql := "INSERT INTO event(eventid, ieventid, event_name, period, starttime, endtime, noentry,"
		sql += " intervalmin, modmin, modsec, "
		sql += " Fromorder, Toorder, Resethh, Resetmm, Nobasis, Maxdsp, Cmap, target, maxpoint "
		sql += ") VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
		log.Printf("db.Prepare(sql)\n")
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("error InsertEventInf() (INSERT/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		log.Printf("row.Exec()\n")
		_, err = stmt.Exec(
			(*eventinf).Event_ID,
			(*eventinf).I_Event_ID,
			(*eventinf).Event_name,
			(*eventinf).Period,
			(*eventinf).Start_time,
			(*eventinf).End_time,
			(*eventinf).NoEntry,
			(*eventinf).Intervalmin,
			(*eventinf).Modmin,
			(*eventinf).Modsec,
			(*eventinf).Fromorder,
			(*eventinf).Toorder,
			(*eventinf).Resethh,
			(*eventinf).Resetmm,
			(*eventinf).Nobasis,
			(*eventinf).Maxdsp,
			(*eventinf).Cmap,
			(*eventinf).Target,
			(*eventinf).Maxpoint,
		)

		if err != nil {
			log.Printf("error InsertEventInf() (INSERT/Exec) err=%s\n", err.Error())
			status = -2
		}
	} else {
		status = 1
	}

	return
}

func UpdateEventInf(eventinf *Event_Inf) (
	status int,
) {

	if _, _, status = SelectEventNoAndName((*eventinf).Event_ID); status == 0 {
		sql := "Update event set "
		sql += " ieventid=?,"
		sql += " event_name=?,"
		sql += " period=?,"
		sql += " starttime=?,"
		sql += " endtime=?,"
		sql += " noentry=?,"
		sql += " intervalmin=?,"
		sql += " modmin=?,"
		sql += " modsec=?,"
		sql += " Fromorder=?,"
		sql += " Toorder=?,"
		sql += " Resethh=?,"
		sql += " Resetmm=?,"
		sql += " Nobasis=?,"
		sql += " Target=?,"
		sql += " Maxdsp=?, "
		sql += " cmap=?, "
		sql += " maxpoint=? "
		//	sql += " where eventno = ?"
		sql += " where eventid = ?"
		log.Printf("db.Prepare(sql)\n")
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("UpdateEventInf() error (Update/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		log.Printf("row.Exec()\n")
		_, err = stmt.Exec(
			(*eventinf).I_Event_ID,
			(*eventinf).Event_name,
			(*eventinf).Period,
			(*eventinf).Start_time,
			(*eventinf).End_time,
			(*eventinf).NoEntry,
			(*eventinf).Intervalmin,
			(*eventinf).Modmin,
			(*eventinf).Modsec,
			(*eventinf).Fromorder,
			(*eventinf).Toorder,
			(*eventinf).Resethh,
			(*eventinf).Resetmm,
			(*eventinf).Nobasis,
			(*eventinf).Target,
			(*eventinf).Maxdsp,
			(*eventinf).Cmap,
			(*eventinf).Maxpoint,
			(*eventinf).Event_ID,
		)

		if err != nil {
			log.Printf("error UpdateEventInf() (update/Exec) err=%s\n", err.Error())
			status = -2
		}
	} else {
		status = 1
	}

	return
}

func InsertRoomInf(eventid string, roominfolist *RoomInfoList) {

	//	log.Printf("***** InsertRoomInf() ***********  NoRoom=%d\n", len(*roominfolist))
	tnow := time.Now().Truncate(time.Second)
	for i := 0; i < len(*roominfolist); i++ {
		InsertIntoOrUpdateUser(tnow, eventid, (*roominfolist)[i])
		status := InsertIntoEventUser(i, eventid, (*roominfolist)[i])
		if status == 0 {
			log.Printf("   ** Update RoomInf() ***********  i=%d, userno=%d\n", i, (*roominfolist)[i].Userno)
			(*roominfolist)[i].Status = "更新"
			(*roominfolist)[i].Statuscolor = "black"
		} else if status == 1 {
			log.Printf("   ** Insert RoomInf() ***********  i=%d\n", i)
			(*roominfolist)[i].Status = "新規"
			(*roominfolist)[i].Statuscolor = "green"
		} else {
			(*roominfolist)[i].Status = "エラー"
			(*roominfolist)[i].Statuscolor = "red"
		}
	}
	log.Printf("***** end of InsertRoomInf() ***********\n")
}

func InsertIntoOrUpdateUser(tnow time.Time, eventid string, roominf RoomInfo) (status int) {

	status = 0

	isnew := false

	userno, _ := strconv.Atoi(roominf.ID)
	//	log.Printf("  *** InsertIntoOrUpdateUser() *** userno=%d\n", userno)

	nrow := 0
	err := Db.QueryRow("select count(*) from user where userno =" + roominf.ID).Scan(&nrow)

	if err != nil {
		log.Printf("select count(*) from user ... err=[%s]\n", err.Error())
		status = -1
		return
	}

	name := ""
	genre := ""
	rank := ""
	nrank := ""
	prank := ""
	level := 0
	followers := 0
	fans := -1
	fans_lst := -1

	if nrow == 0 {

		isnew = true

		log.Printf("   ** insert into userhistory(*new*) userno=%d rank=<%s> nrank=<%s> prank=<%s> level=%d, followers=%d, fans=%d, fans_lst=%d\n",
			userno, roominf.Rank, roominf.Nrank, roominf.Prank, roominf.Level, roominf.Followers, fans, fans_lst)

		sql := "INSERT INTO user(userno, userid, user_name, longname, shortname, genre, `rank`, nrank, prank, level, followers, fans, fans_lst, ts, currentevent)"
		sql += " VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

		//	log.Printf("sql=%s\n", sql)
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("InsertIntoOrUpdateUser() error() (INSERT/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		lenid := len(roominf.ID)
		_, err = stmt.Exec(
			userno,
			roominf.Account,
			roominf.Name,
			//	roominf.ID,
			roominf.Name,
			roominf.ID[lenid-2:lenid],
			roominf.Genre,
			roominf.Rank,
			roominf.Nrank,
			roominf.Prank,
			roominf.Level,
			roominf.Followers,
			roominf.Fans,
			roominf.Fans_lst,
			tnow,
			eventid,
		)

		if err != nil {
			log.Printf("error(InsertIntoOrUpdateUser() INSERT/Exec) err=%s\n", err.Error())
			//	status = -2
			_, err = stmt.Exec(
				userno,
				roominf.Account,
				roominf.Account,
				roominf.ID,
				roominf.ID[lenid-2:lenid],
				roominf.Genre,
				roominf.Rank,
				roominf.Nrank,
				roominf.Prank,
				roominf.Level,
				roominf.Followers,
				roominf.Fans,
				roominf.Fans_lst,
				tnow,
				eventid,
			)
			if err != nil {
				log.Printf("error(InsertIntoOrUpdateUser() INSERT/Exec) err=%s\n", err.Error())
				status = -2
			}
		}
	} else {

		sql := "select user_name, genre, `rank`, nrank, prank, level, followers, fans, fans_lst from user where userno = ?"
		err = Db.QueryRow(sql, userno).Scan(&name, &genre, &rank, &nrank, &prank, &level, &followers, &fans, &fans_lst)
		if err != nil {
			log.Printf("err=[%s]\n", err.Error())
			status = -1
		}
		//	log.Printf("current userno=%d name=%s, nrank=%s, prank=%s level=%d, followers=%d\n", userno, name, nrank, prank, level, followers)

		if roominf.Genre != genre ||
			roominf.Rank != rank ||
			//	roominf.Nrank != nrank ||
			//	roominf.Prank != prank ||
			roominf.Level != level ||
			roominf.Followers != followers ||
			roominf.Fans != fans {

			isnew = true

			log.Printf("   ** insert into userhistory(*changed*) userno=%d level=%d, followers=%d, fans=%d\n",
				userno, roominf.Level, roominf.Followers, roominf.Fans)
			sql := "update user set userid=?,"
			sql += "user_name=?,"
			sql += "genre=?,"
			sql += "`rank`=?,"
			sql += "nrank=?,"
			sql += "prank=?,"
			sql += "level=?,"
			sql += "followers=?,"
			sql += "fans=?,"
			sql += "fans_lst=?,"
			sql += "ts=?,"
			sql += "currentevent=? "
			sql += "where userno=?"
			stmt, err := Db.Prepare(sql)

			if err != nil {
				log.Printf("InsertIntoOrUpdateUser() error(Update/Prepare) err=%s\n", err.Error())
				status = -1
				return
			}
			defer stmt.Close()

			_, err = stmt.Exec(
				roominf.Account,
				roominf.Name,
				roominf.Genre,
				roominf.Rank,
				roominf.Nrank,
				roominf.Prank,
				roominf.Level,
				roominf.Followers,
				roominf.Fans,
				roominf.Fans_lst,
				tnow,
				eventid,
				roominf.ID,
			)

			if err != nil {
				log.Printf("error(InsertIntoOrUpdateUser() Update/Exec) err=%s\n", err.Error())
				status = -2
			}
		}
		/* else {
			//	log.Printf("not insert into userhistory(*same*) userno=%d level=%d, followers=%d\n", userno, roominf.Level, roominf.Followers)
		}
		*/

	}

	if isnew {
		sql := "INSERT INTO userhistory(userno, user_name, genre, `rank`, nrank, prank, level, followers, fans, fans_lst, ts)"
		sql += " VALUES(?,?,?,?,?,?,?,?,?,?,?)"
		//	log.Printf("sql=%s\n", sql)
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("error(INSERT into userhistory/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(
			userno,
			roominf.Name,
			roominf.Genre,
			roominf.Rank,
			roominf.Nrank,
			roominf.Prank,
			roominf.Level,
			roominf.Followers,
			roominf.Fans,
			roominf.Fans_lst,
			tnow,
		)

		if err != nil {
			log.Printf("error(Insert Into into userhistory INSERT/Exec) err=%s\n", err.Error())
			//	status = -2
			_, err = stmt.Exec(
				userno,
				roominf.Account,
				roominf.Genre,
				roominf.Rank,
				roominf.Nrank,
				roominf.Prank,
				roominf.Level,
				roominf.Followers,
				roominf.Fans,
				roominf.Fans_lst,
				tnow,
			)
			if err != nil {
				log.Printf("error(Insert Into into userhistory INSERT/Exec) err=%s\n", err.Error())
				status = -2
			}
		}

	}

	return

}
func InsertIntoEventUser(i int, eventid string, roominf RoomInfo) (status int) {

	status = 0

	userno, _ := strconv.Atoi(roominf.ID)

	nrow := 0
	/*
		sql := "select count(*) from eventuser where "
		sql += "userno =" + roominf.ID + " and "
		//	sql += "eventno = " + fmt.Sprintf("%d", eventno)
		sql += "eventid = " + eventid
		//	log.Printf("sql=%s\n", sql)
		err := Db.QueryRow(sql).Scan(&nrow)
	*/
	sql := "select count(*) from eventuser where userno =? and eventid = ?"
	err := Db.QueryRow(sql, roominf.ID, eventid).Scan(&nrow)

	if err != nil {
		log.Printf("select count(*) from user ... err=[%s]\n", err.Error())
		status = -1
		return
	}

	Colorlist := Colorlist2
	if Event_inf.Cmap == 1 {
		Colorlist = Colorlist1
	}

	if nrow == 0 {
		//	log.Printf("  =====Insert into eventuser userno=%d, eventid=%s\n", userno, eventid)
		sql := "INSERT INTO eventuser(eventid, userno, istarget, graph, color, iscntrbpoints, point) VALUES(?,?,?,?,?,?,?)"
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("error(INSERT/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		//	if i < 10 {
		_, err = stmt.Exec(
			eventid,
			userno,
			"Y",
			"Y",
			Colorlist[i%len(Colorlist)].Name,
			"N",
			roominf.Point,
		)
		/*
			} else {
				_, err = stmt.Exec(
					eventid,
					userno,
					"Y",	//	"N"から変更する＝順位に関わらず獲得ポイントデータを取得する。
					"N",
					Colorlist[i%len(Colorlist)].Name,
					"N",
					roominf.Point,
				)
			}
		*/

		if err != nil {
			log.Printf("error(InsertIntoOrUpdateUser() INSERT/Exec) err=%s\n", err.Error())
			status = -2
		}
		status = 1
	} else {
		//	log.Printf("  =====Update eventuser userno=%d, eventid=%s\n", userno, eventid)
		sql := "UPDATE eventuser SET istarget=? where eventid=? and userno=?"
		stmt, err := Db.Prepare(sql)
		if err != nil {
			log.Printf("error(UPDATE/Prepare) err=%s\n", err.Error())
			status = -1
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(
			"Y",
			eventid,
			userno,
		)

		if err != nil {
			log.Printf("error(InsertIntoOrUpdateUser() UPDATE/Exec) err=%s\n", err.Error())
			status = -2
		}
	}
	return

}

func GetEventInfAndRoomList(
	eventid string,
	breg int,
	ereg int,
	eventinfo *Event_Inf,
	roominfolist *RoomInfoList,
) (
	status int,
) {

	//	画面からのデータ取得部分は次を参考にしました。
	//		はじめてのGo言語：Golangでスクレイピングをしてみた
	//		https://qiita.com/ryo_naka/items/a08d70f003fac7fb0808

	//	_url := "https://www.showroom-live.com/event/" + EventID
	//	_url = "file:///C:/Users/kohei47/Go/src/EventRoomList03/20210128-1143.html"
	//	_url = "file:20210128-1143.html"

	var doc *goquery.Document
	var err error

	inputmode := "url"
	eventidorfilename := eventid
	maxroom := ereg

	status = 0

	if inputmode == "file" {

		//	ファイルからドキュメントを作成します
		f, e := os.Open(eventidorfilename)
		if e != nil {
			//	log.Fatal(e)
			log.Printf("err=[%s]\n", e.Error())
			status = -1
			return
		}
		defer f.Close()
		doc, err = goquery.NewDocumentFromReader(f)
		if err != nil {
			//	log.Fatal(err)
			log.Printf("err=[%s]\n", err.Error())
			status = -1
			return
		}

		content, _ := doc.Find("head > meta:nth-child(6)").Attr("content")
		content_div := strings.Split(content, "/")
		(*eventinfo).Event_ID = content_div[len(content_div)-1]

	} else {
		//	URLからドキュメントを作成します
		_url := "https://www.showroom-live.com/event/" + eventidorfilename
		/*
			doc, err = goquery.NewDocument(_url)
		*/
		resp, error := http.Get(_url)
		if error != nil {
			log.Printf("GetEventInfAndRoomList() http.Get() err=%s\n", error.Error())
			status = 1
			return
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)

		//	bufstr := buf.String()
		//	log.Printf("%s\n", bufstr)

		//	doc, error = goquery.NewDocumentFromReader(resp.Body)
		doc, error = goquery.NewDocumentFromReader(buf)
		if error != nil {
			log.Printf("GetEventInfAndRoomList() goquery.NewDocumentFromReader() err=<%s>.\n", error.Error())
			status = 1
			return
		}

		(*eventinfo).Event_ID = eventidorfilename
	}
	//	log.Printf(" eventid=%s\n", (*eventinfo).Event_ID)

	cevent_id, exists := doc.Find("#eventDetail").Attr("data-event-id")
	if !exists {
		log.Printf("data-event-id not found. Event_ID=%s\n", (*eventinfo).Event_ID)
		status = -1
		return
	}
	eventinfo.I_Event_ID, _ = strconv.Atoi(cevent_id)

	selector := doc.Find(".detail")
	(*eventinfo).Event_name = selector.Find(".tx-title").Text()
	if (*eventinfo).Event_name == "" {
		log.Printf("Event not found. Event_ID=%s\n", (*eventinfo).Event_ID)
		status = -1
		return
	}
	(*eventinfo).Period = selector.Find(".info").Text()
	eventinfo.Period = strings.Replace(eventinfo.Period, "\u202f", " ", -1)
	period := strings.Split((*eventinfo).Period, " - ")
	if inputmode == "url" {
		(*eventinfo).Start_time, _ = time.Parse("Jan 2, 2006 3:04 PM MST", period[0]+" JST")
		(*eventinfo).End_time, _ = time.Parse("Jan 2, 2006 3:04 PM MST", period[1]+" JST")
	} else {
		(*eventinfo).Start_time, _ = time.Parse("2006/01/02 15:04 MST", period[0]+" JST")
		(*eventinfo).End_time, _ = time.Parse("2006/01/02 15:04 MST", period[1]+" JST")
	}

	(*eventinfo).EventStatus = "BeingHeld"
	if (*eventinfo).Start_time.After(time.Now()) {
		(*eventinfo).EventStatus = "NotHeldYet"
	} else if (*eventinfo).End_time.Before(time.Now()) {
		(*eventinfo).EventStatus = "Over"
	}

	//	イベントに参加しているルームの数を求めます。
	//	参加ルーム数と表示されているルームの数は違うので、ここで取得したルームの数を以下の処理で使うわけではありません。
	SNoEntry := doc.Find("p.ta-r").Text()
	fmt.Sscanf(SNoEntry, "%d", &((*eventinfo).NoEntry))
	log.Printf("[%s]\n[%s] [%s] (*event).EventStatus=%s NoEntry=%d\n",
		(*eventinfo).Event_name,
		(*eventinfo).Start_time.Format("2006/01/02 15:04 MST"),
		(*eventinfo).End_time.Format("2006/01/02 15:04 MST"),
		(*eventinfo).EventStatus, (*eventinfo).NoEntry)
	log.Printf("breg=%d ereg=%d\n", breg, ereg)

	//	eventno, _, _ := SelectEventNoAndName(eventidorfilename)
	//	log.Printf(" eventno=%d\n", eventno)
	//	(*eventinfo).Event_no = eventno

	//	抽出したルームすべてに対して処理を繰り返す(が、イベント開始後の場合の処理はルーム数がbreg、eregの範囲に限定）
	//	イベント開始前のときはすべて取得し、ソートしたあてで範囲を限定する）
	doc.Find(".listcardinfo").EachWithBreak(func(i int, s *goquery.Selection) bool {
		//	log.Printf("i=%d\n", i)
		if (*eventinfo).Start_time.Before(time.Now()) {
			if i < breg-1 {
				return true
			}
			if i == maxroom {
				return false
			}
		}

		var roominfo RoomInfo

		roominfo.Name = s.Find(".listcardinfo-main-text").Text()

		spoint1 := strings.Split(s.Find(".listcardinfo-sub-single-right-text").Text(), ": ")

		var point int64
		if spoint1[0] != "" {
			spoint2 := strings.Split(spoint1[1], "pt")
			fmt.Sscanf(spoint2[0], "%d", &point)

		} else {
			point = -1
		}
		roominfo.Point = int(point)

		ReplaceString := ""

		selection_c := s.Find(".listcardinfo-menu")

		account, _ := selection_c.Find(".room-url").Attr("href")
		if inputmode == "file" {
			ReplaceString = "https://www.showroom-live.com/"
		} else {
			ReplaceString = "/r/"
		}
		roominfo.Account = strings.Replace(account, ReplaceString, "", -1)
		roominfo.Account = strings.Replace(roominfo.Account, "/", "", -1)

		roominfo.ID, _ = selection_c.Find(".js-follow-btn").Attr("data-room-id")
		roominfo.Userno, _ = strconv.Atoi(roominfo.ID)

		*roominfolist = append(*roominfolist, roominfo)

		//	log.Printf("%11s %-20s %-10s %s\n",
		//		humanize.Comma(int64(roominfo.Point)), roominfo.Account, roominfo.ID, roominfo.Name)
		return true

	})

	(*eventinfo).NoRoom = len(*roominfolist)

	log.Printf(" GetEventInfAndRoomList() len(*roominfolist)=%d\n", len(*roominfolist))

	return
}
