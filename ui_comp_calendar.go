/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this db except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"strconv"
	"time"
)

func (ui *Ui) UiCalendar_MonthText(month int) string {
	switch month {
	case 1:
		return ui.trns.JANUARY
	case 2:
		return ui.trns.FEBRUARY
	case 3:
		return ui.trns.MARCH
	case 4:
		return ui.trns.APRIL
	case 5:
		return ui.trns.MAY
	case 6:
		return ui.trns.JUNE
	case 7:
		return ui.trns.JULY
	case 8:
		return ui.trns.AUGUST
	case 9:
		return ui.trns.SEPTEMBER
	case 10:
		return ui.trns.OCTOBER
	case 11:
		return ui.trns.NOVEMBER
	case 12:
		return ui.trns.DECEMBER
	}
	return ""
}

func (ui *Ui) UiCalendar_DayTextFull(day int) string {

	switch day {
	case 1:
		return ui.trns.MONDAY
	case 2:
		return ui.trns.TUESDAY
	case 3:
		return ui.trns.WEDNESDAY
	case 4:
		return ui.trns.THURSDAY
	case 5:
		return ui.trns.FRIDAY
	case 6:
		return ui.trns.SATURDAY
	case 7:
		return ui.trns.SUNDAY
	}
	return ""
}

func (ui *Ui) UiCalendar_DayTextShort(day int) string {

	switch day {
	case 1:
		return ui.trns.MON
	case 2:
		return ui.trns.TUE
	case 3:
		return ui.trns.WED
	case 4:
		return ui.trns.THU
	case 5:
		return ui.trns.FRI
	case 6:
		return ui.trns.SAT
	case 7:
		return ui.trns.SUN
	}
	return ""
}

type SADate struct {
	Year, Month, Day     int
	Hour, Minute, Second int

	WeekDay        int //sun=0, mon=1, etc.
	ZoneOffset_sec int
	ZoneName       string
	IsDST          bool
}

func (d *SADate) GetWeekDay(dateFormat int) int {
	week := d.WeekDay
	if dateFormat != 1 {
		//not "us"
		week -= 1
		if week < 0 {
			week = 6
		}
	}
	return week
}

func (a *SADate) CmpYMD(b *SADate) bool {
	return a.Year == b.Year && a.Month == b.Month && a.Day == b.Day
}
func (a *SADate) CmpYMDHMS(b *SADate) bool {
	return a.CmpYMD(b) && a.Hour == b.Hour && a.Minute == b.Minute && a.Second == b.Second
}

func SA_InfoGetDateFromTime(unixTime int64) SADate {
	tm := time.Unix(unixTime, 0)
	zoneName, zoneOffset_sec := tm.Zone()

	var d SADate
	d.Year = tm.Year()
	d.Month = int(tm.Month())
	d.Day = tm.Day()
	d.Hour = tm.Hour()
	d.Minute = tm.Minute()
	d.Second = tm.Second()

	d.WeekDay = int(tm.Weekday())

	d.ZoneOffset_sec = zoneOffset_sec
	d.ZoneName = zoneName
	d.IsDST = tm.IsDST()

	return d
}

func SA_InfoGetTimeFromDate(date *SADate) int64 {
	tm := time.Date(date.Year, time.Month(date.Month), date.Day, date.Hour, date.Minute, date.Second, 0, time.Local)
	return tm.Unix()
}

func SA_InfoAddDate(unixTime int64, add_years, add_months, add_days int) int64 {
	tm := time.Unix(unixTime, 0)
	return tm.AddDate(add_years, add_months, add_days).Unix()
}

func UiCalendar_GetStartDay(unix_sec int64) int64 {
	d := SA_InfoGetDateFromTime(unix_sec)
	return unix_sec - int64(d.Hour)*3600 - int64(d.Minute)*60 - int64(d.Second)
}

func UiCalendar_GetStartWeek(unix_sec int64, dateFormat int) int64 {
	unix_sec = UiCalendar_GetStartDay(unix_sec)

	d := SA_InfoGetDateFromTime(unix_sec)
	weekDay := d.GetWeekDay(dateFormat) //možná dát dateFormat, také do SADate? ...

	return SA_InfoAddDate(unix_sec, 0, 0, -weekDay)
}

func UiCalendar_GetStartMonth(unix_sec int64) int64 {
	d := SA_InfoGetDateFromTime(unix_sec)
	d.Day = 1
	return SA_InfoGetTimeFromDate(&d)
}

func (ui *Ui) GetTextDate(unix_sec int64) string {

	dd := SA_InfoGetDateFromTime(unix_sec)

	switch ui.win.io.ini.DateFormat {
	case 0: //eu
		return fmt.Sprintf("%d/%d/%d", dd.Day, dd.Month, dd.Year)

	case 1: //us
		return fmt.Sprintf("%d/%d/%d", dd.Month, dd.Day, dd.Year)

	case 2: //iso
		return fmt.Sprintf("%d-%02d-%02d", dd.Year, dd.Month, dd.Day)

	case 3: //text
		return fmt.Sprintf("%s %d, %d", ui.UiCalendar_MonthText(dd.Month), dd.Day, dd.Year)

	case 4: //2base
		return fmt.Sprintf("%d %d-%d", dd.Year, dd.Month, dd.Day)
	}

	return ""
}
func UiCalendar_GetTextTime(unix_sec int64) string {
	d := SA_InfoGetDateFromTime(unix_sec)
	return fmt.Sprintf("%d:%d", d.Hour, d.Minute)
}

func (ui *Ui) GetTextDateTime(unix_sec int64) string {
	return ui.GetTextDate(unix_sec) + " " + UiCalendar_GetTextTime(unix_sec)
}

func (ui *Ui) UiCalendar_GetMonthYear(unix_sec int64) string {
	d := SA_InfoGetDateFromTime(unix_sec)
	return ui.UiCalendar_MonthText(d.Month) + " " + strconv.Itoa(d.Year)
}

func UiCalendar_GetYear(unix_sec int64) string {
	d := SA_InfoGetDateFromTime(unix_sec)
	return strconv.Itoa(d.Year)
}

/*func (ui *Ui) Comp_calendar(x, y, w, h int, value *int64, page *int64) bool {
	ui.Div_start(x, y, w, h)
	changed := ui.Comp_Calendar_s(value, page)
	ui.Div_end()

	return changed
}*/

func (ui *Ui) Comp_Calendar(value *int64, page *int64) bool {

	old_value := *value
	format := ui.win.io.ini.DateFormat

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(1, 100)

	//head
	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(0, 2)
		ui.Div_colMax(1, 100)

		if ui.Comp_buttonLight(0, 0, 1, 1, ui.trns.TODAY, ui.GetTextDate(int64(OsTime())), true) > 0 {
			*page = int64(OsTime())
			*value = int64(OsTime())
		}

		ui.Comp_text(1, 0, 1, 1, "##"+ui.UiCalendar_GetMonthYear(*page), 1)

		if ui.Comp_buttonLight(2, 0, 1, 1, "<", "", true) > 0 {
			*page = SA_InfoAddDate(*page, 0, -1, 0)
		}

		if ui.Comp_buttonLight(3, 0, 1, 1, ">", "", true) > 0 {
			*page = SA_InfoAddDate(*page, 0, 1, 0)
		}
	}
	ui.Div_end()

	//days
	ui.Div_start(0, 1, 1, 1)
	{
		for x := 0; x < 7; x++ {
			ui.Div_col(x, 0.9)
			ui.Div_colMax(x, 2)
		}
		for y := 0; y < 7; y++ {
			ui.Div_row(y, 0.9)
			ui.Div_rowMax(y, 2)
		}

		//fix page(need to start with day 1)
		*page = UiCalendar_GetStartMonth(*page)

		//--Day names(short)--
		if format == 1 {
			//"us"
			ui.Comp_text(0, 0, 1, 1, ui.UiCalendar_DayTextShort(7), 1)
			for x := 1; x < 7; x++ {
				ui.Comp_text(x, 0, 1, 1, ui.UiCalendar_DayTextShort(x), 1)
			}
		} else {
			for x := 1; x < 8; x++ {
				ui.Comp_text(x-1, 0, 1, 1, ui.UiCalendar_DayTextShort(x), 1)
			}
		}

		//--Week days--
		today := SA_InfoGetDateFromTime(int64(OsTime()))
		value_dtt := SA_InfoGetDateFromTime(*value)
		curr_month := SA_InfoGetDateFromTime(*page).Month
		dt := UiCalendar_GetStartWeek(*page, format)

		for y := 0; y < 6; y++ {
			for x := 0; x < 7; x++ {
				showBack := false
				//backCd := SACd_B //GetThemeCd()
				fade := false //default

				dtt := SA_InfoGetDateFromTime(dt)
				isDayToday := today.CmpYMD(&dtt)        //CmpDates(dtt.Unix(), now)
				isDaySelected := value_dtt.CmpYMD(&dtt) //CmpDates(dtt.Unix(), *value)
				isDayInMonth := dtt.Month == curr_month

				if isDaySelected && isDayInMonth { //selected day
					showBack = true
					//backCd = SACd_P
				}
				//if isDayToday {
				//backCd = SACd_T
				//}
				if !isDayInMonth { //is day in current month
					fade = true
				}

				clicked := false
				if isDayToday {
					clicked = ui.Comp_buttonOutlinedFade(x, 1+y, 1, 1, strconv.Itoa(dtt.Day), "", true, showBack, fade) > 0
				} else {
					clicked = ui.Comp_buttonTextFade(x, 1+y, 1, 1, strconv.Itoa(dtt.Day), "", "", true, showBack, fade) > 0
				}
				if clicked {
					*value = dt
					*page = *value
				}

				dt = SA_InfoAddDate(dt, 0, 0, 1) //add day
			}
		}
	}
	ui.Div_end()

	return old_value != *value
}

func (ui *Ui) Comp_CalendarDataPicker(date_unix int64, divName string) int64 {
	//ui.Div_colMax(0, 3)
	ui.Div_colMax(0, 15)

	//SA_Text(name).Show(0, 0, 1, 1)

	hm_over := date_unix - UiCalendar_GetStartDay(date_unix)

	//date
	if ui.Comp_button(0, 0, 1, 1, ui.GetTextDate(date_unix), "", true) > 0 {
		ui.Dialog_open("DateTimePicker_"+divName, 1)
		ui.date_page = int64(OsTime())
	}

	if ui.Dialog_start("DateTimePicker_" + divName) {
		if ui.Comp_Calendar(&date_unix, &ui.date_page) {
			//keep old hour/minute
			date_unix = UiCalendar_GetStartDay(date_unix) //date_unix % (24 * 3600)
			date_unix += hm_over
		}
		ui.Dialog_end()
	}

	//time
	tm := SA_InfoGetDateFromTime(date_unix)
	hour := tm.Hour
	minute := tm.Minute

	editChanged := false
	_, _, _, fnshd, _ := ui.Comp_editbox(2, 0, 1, 1, &hour, 0, "", ui.trns.HOUR, false, true, true)
	if fnshd {
		if hour < 0 {
			hour = 0
		}
		if hour > 23 {
			hour = 23
		}
		editChanged = true
	}

	ui.Comp_text(3, 0, 1, 1, ":", 1)

	_, _, _, fnshd, _ = ui.Comp_editbox(4, 0, 1, 1, &minute, 0, "", ui.trns.MINUTE, false, true, true)
	if fnshd {
		if minute < 0 {
			minute = 0
		}
		if minute > 59 {
			minute = 59
		}
		editChanged = true
	}

	//modify hour/minute
	if editChanged {
		date_unix = SA_InfoGetTimeFromDate(&SADate{Year: tm.Year, Month: tm.Month, Day: tm.Day, Hour: hour, Minute: minute}) //- int64(store.timezone)
		//tm = GetTimeSt(date_unix)
		//date_unix = time.Date(tm.Year(), tm.Month(), tm.Day(), hour, minute, 0, 0, tm.Location()).Unix()
	}

	return date_unix
}
