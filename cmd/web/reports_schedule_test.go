package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) InsertReportSchedule(_ context.Context, arg db.InsertReportScheduleParams) (db.ReportSchedule, error) {
	if f.rsNextID == 0 {
		f.rsNextID = 1
	}
	rs := db.ReportSchedule{
		ID: f.rsNextID, Name: arg.Name, Sections: arg.Sections,
		Cadence: arg.Cadence, Format: arg.Format,
		DeliveryTarget: arg.DeliveryTarget, ChannelID: arg.ChannelID,
		CreatedBy: arg.CreatedBy,
	}
	f.rsNextID++
	f.reportSchedules = append(f.reportSchedules, rs)
	return rs, nil
}

func (f *fakeStore) GetReportSchedule(_ context.Context, id int64) (db.ReportSchedule, error) {
	for _, rs := range f.reportSchedules {
		if rs.ID == id {
			return rs, nil
		}
	}
	return db.ReportSchedule{}, pgx.ErrNoRows
}

func (f *fakeStore) UpdateReportSchedule(_ context.Context, arg db.UpdateReportScheduleParams) (db.ReportSchedule, error) {
	for i, rs := range f.reportSchedules {
		if rs.ID != arg.ID {
			continue
		}
		rs.Name = arg.Name
		rs.Sections = arg.Sections
		rs.Cadence = arg.Cadence
		rs.Format = arg.Format
		rs.DeliveryTarget = arg.DeliveryTarget
		rs.ChannelID = arg.ChannelID
		f.reportSchedules[i] = rs
		return rs, nil
	}
	return db.ReportSchedule{}, pgx.ErrNoRows
}

func (f *fakeStore) DeleteReportSchedule(_ context.Context, id int64) error {
	out := f.reportSchedules[:0]
	for _, rs := range f.reportSchedules {
		if rs.ID != id {
			out = append(out, rs)
		}
	}
	f.reportSchedules = out
	return nil
}

func (f *fakeStore) NextReportDeliveryNo(_ context.Context, scheduleID int64) (int32, error) {
	var max int32
	for _, d := range f.reportDeliveries {
		if d.ScheduleID == scheduleID && d.DeliveryNo > max {
			max = d.DeliveryNo
		}
	}
	return max + 1, nil
}

func (f *fakeStore) InsertReportDelivery(_ context.Context, arg db.InsertReportDeliveryParams) (db.ReportDelivery, error) {
	f.rdNextID++
	d := db.ReportDelivery{
		ID:          f.rdNextID,
		ScheduleID:  arg.ScheduleID,
		PeriodStart: arg.PeriodStart,
		PeriodEnd:   arg.PeriodEnd,
		DeliveryNo:  arg.DeliveryNo,
		GeneratedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		DeliveredAt: arg.DeliveredAt,
		State:       arg.State,
	}
	f.reportDeliveries = append(f.reportDeliveries, d)
	return d, nil
}
