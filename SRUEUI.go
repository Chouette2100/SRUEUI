/*
!
Copyright © 2023 chouette.21.00@gmail.com
Released under the MIT license
https://opensource.org/licenses/mit-license.php
*/
package main

import (
	"fmt"
	"log"

	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	//	"bufio"
	"bytes"
	"io"
	"os"

	//	"runtime"

	//	"encoding/json"

	//	"html/template"
	"net/http"

	//	"database/sql"
	//	_ "github.com/go-sql-driver/mysql"

	"github.com/PuerkitoBio/goquery"

	//	svg "github.com/ajstarks/svgo/float"

	"github.com/dustin/go-humanize"

	//	scl "UpdateUserInf/ShowroomCGIlib"
	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srapi"

	"SRUEUI/srapilc"
	"SRUEUI/srdblc"
)

/*
	イベントのユーザ情報を更新する。

	Ver.000AA000 新規作成
	Ver.000AB000 イベント開始直後と終了直前のデータ取得の頻度を上げる。

*/

const Version = "000AB000"

func GetAndInsertEventRoomInfo(
	client *http.Client,
	eventid string,
	breg int,
	ereg int,
	eventinfo *srdblc.Event_Inf,
	roominfolist *srdblc.RoomInfoList,
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
		status = srdblc.GetEventInfAndRoomList(eventid, breg, ereg, eventinfo, roominfolist)
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
			srdblc.UpdateEventuserSetPoint(eventid, (*roominfolist)[i].ID, point)
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
		srdblc.SortByFollowers = true
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
	status = srdblc.InsertEventInf(eventinfo)

	if status == 1 {
		log.Println("InsertEventInf() returned 1.")
		srdblc.UpdateEventInf(eventinfo)
		status = 0
	}
	log.Println("GetAndInsertEventRoomInfo() after InsertEventIinf() or UpdateEventInf")
	log.Println(*eventinfo)

	_, _, status = srdblc.SelectEventNoAndName(eventid)

	if status == 0 {
		//	InsertRoomInf(eventno, eventid, roominfolist)
		srdblc.InsertRoomInf(eventid, roominfolist)
	}

	return
}

func GetEventInfAndRoomListBR(
	client *http.Client,
	eventid string,
	breg int,
	ereg int,
	eventinfo *srdblc.Event_Inf,
	roominfolist *srdblc.RoomInfoList,
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

		var roominfo srdblc.RoomInfo

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

func main() {

	var eventinf srdblc.Event_Inf
	var roominfolist srdblc.RoomInfoList

	//	ログ出力を設定する
	logfilename := Version + "_" + srdblc.Version  + "_" + srapilc.Version + "_" + time.Now().Format("20060102") + ".txt"
	logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannnot open logfile: " + logfilename + err.Error())
	}
	defer logfile.Close()
	//	log.SetOutput(logfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	//	サーバー設定を読み込む
	exerr := exsrapi.LoadConfig("ServerConfig.yml", &srdblc.Dbconfig)
	if exerr != nil {
		log.Printf("LoadConfig: %s\n", exerr.Error())
		return
	}

	dbconfig := srdblc.Dbconfig

	log.Printf("%+v\n", *dbconfig)

	log.Printf("\n")
	log.Printf("\n")
	log.Printf("********** Dbhost=<%s> Dbname = <%s> Dbuser = <%s> Dbpw = <%s>\n", (*dbconfig).Dbhost, (*dbconfig).Dbname, (*dbconfig).Dbuser, (*dbconfig).Dbpw)

	//	データベースとの接続をオープンする。
	status := srdblc.OpenDb()
	if status != 0 {
		log.Printf("Database error.\n")
		return
	}
	defer srdblc.Db.Close()

	/*
		if len(os.Args) != 2 {
			log.Printf("usage: %s eventid\n", os.Args[0])
			os.Exit(1)
		}
	*/

	//      cookiejarがセットされたHTTPクライアントを作る
	client, jar, err := exsrapi.CreateNewClient("anonymous")
	if err != nil {
		log.Printf("CreateNewClient() returned error %s\n", err.Error())
		return
	}
	//      すべての処理が終了したらcookiejarを保存する。
	defer jar.Save() //	忘れずに！

	//	現在開催中のイベントのリストを得る
	//	（開始前のものは含まない）
	eventlist, status := srdblc.SelectCurEventList()
	if status != 0 {
		log.Printf("status=%d.\n", status)
		os.Exit(1)
	}

	for _, event := range eventlist {
		log.Printf(" eventid=[%s] eventname=%s.\n", event.Event_ID, event.Event_name)
		qnow := time.Now().Minute()/15	// このモジュールが15分に一度起動されることを前提としている。
		hs := time.Since(event.Start_time).Hours()
		he := time.Until(event.End_time).Hours()
		h := math.Min(hs,he)
		if h > 48.0 && qnow != 0 {
				//	開始2日以後かつ終了2日以前の場合は１時間に1回データを取得する。
				continue
		}
		if h > 6.0 && qnow % 2 != 0 {
				//	開始6時間以後かつ終了6時間以前の場合は１時間に2回データを取得する。
				continue
		}

		eventinf, status = srdblc.SelectEventInf(event.Event_ID)
		if status != 0 {
			log.Printf(" eventid=[%s] status=%d.\n", event.Event_ID, status)
			os.Exit(1)
		}
		roominfolist = srdblc.RoomInfoList{}
		GetAndInsertEventRoomInfo(client, event.Event_ID, event.Fromorder, event.Toorder, &eventinf, &roominfolist)
	}

}
