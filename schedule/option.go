package schedule

import (
	"fmt"
	"time"

	"github.com/naufalfmm/angle"
	"github.com/naufalfmm/angle/trig"
	"github.com/naufalfmm/moslem-salat-schedule/consts"
	higherLatEnum "github.com/naufalfmm/moslem-salat-schedule/enum/higherLat"
	mazhabEnum "github.com/naufalfmm/moslem-salat-schedule/enum/mazhab"
	periodicalEnum "github.com/naufalfmm/moslem-salat-schedule/enum/periodical"
	roundingTimeOptionEnum "github.com/naufalfmm/moslem-salat-schedule/enum/roundingTimeOption"
	salatEnum "github.com/naufalfmm/moslem-salat-schedule/enum/salat"
	sunZenithEnum "github.com/naufalfmm/moslem-salat-schedule/enum/sunZenith"
	"github.com/naufalfmm/moslem-salat-schedule/err"
	"github.com/naufalfmm/moslem-salat-schedule/option"
	"github.com/naufalfmm/moslem-salat-schedule/utils/salatHighAltitude"
	"github.com/naufalfmm/moslem-salat-schedule/utils/sunPositions"
)

type opt struct {
	dateStart  time.Time
	dateEnd    time.Time
	periodical periodicalEnum.Periodical

	latitude    angle.Angle
	longitude   angle.Angle
	elevation   float64
	timezoneLoc *time.Location

	fajrZenith     angle.Angle
	ishaZenith     angle.Angle
	ishaZenithType sunZenithEnum.IshaZenithType

	mazhab               mazhabEnum.Mazhab
	higherLatitudeMethod higherLatEnum.HigherLat

	roundingTimeOption roundingTimeOptionEnum.RoundingTimeOption

	sunPositions sunPositions.SunPositions
}

func (o *opt) SetDateRange(dateStart, dateEnd time.Time) option.Option {
	o.dateStart = dateStart
	o.dateEnd = dateEnd
	o.periodical = periodicalEnum.GetByDateRange(dateStart, dateEnd)

	o.sunPositions = nil

	return o
}

func (o *opt) SetNow() option.Option {
	return o.SetDateRange(time.Now(), time.Now())
}

func (o *opt) SetDatePeriodical(dateStart time.Time, periodical periodicalEnum.Periodical) option.Option {
	o.dateStart, o.dateEnd = periodical.GetDateRange(dateStart)
	o.periodical = periodical

	o.sunPositions = nil

	return o
}

func (o *opt) SetPeriodical(periodical periodicalEnum.Periodical) option.Option {
	if o.dateStart.IsZero() {
		o.dateStart = time.Now()
	}

	return o.SetDatePeriodical(o.dateStart, periodical)
}

func (o *opt) SetLatitudeLongitude(latitude, longitude angle.Angle) option.Option {
	o.latitude = latitude
	o.longitude = longitude

	return o
}

func (o *opt) SetElevation(elevation float64) option.Option {
	o.elevation = elevation

	return o
}

func (o *opt) SetMazhab(mazhab mazhabEnum.Mazhab) option.Option {
	o.mazhab = mazhab

	return o
}

func (o *opt) SetHigherLatitudeMethod(higherLatMethod higherLatEnum.HigherLat) option.Option {
	o.higherLatitudeMethod = higherLatMethod

	return o
}

func (o *opt) SetRoundingTimeOption(roundingTimeOpt roundingTimeOptionEnum.RoundingTimeOption) option.Option {
	o.roundingTimeOption = roundingTimeOpt

	return o
}

func (o *opt) SetTimezoneOffset(timezoneOffset float64) option.Option {
	angTime := angle.NewDegreeFromFloat(timezoneOffset)

	negStr := ""
	if angTime.IsNegative() {
		negStr = "-"
	}

	o.timezoneLoc = time.FixedZone(fmt.Sprintf("%s%s", negStr, angTime.Abs().ToTime().Format("0304")), int(timezoneOffset*consts.OffsetTimezone))

	return o
}

func (o *opt) SetTimezone(timezone *time.Location) option.Option {
	o.timezoneLoc = timezone

	return o
}

func (o *opt) SetFajrIshaZenith(fajrZenith, ishaZenith angle.Angle) option.Option {
	o.fajrZenith = fajrZenith
	o.ishaZenith = ishaZenith
	o.ishaZenithType = sunZenithEnum.Standard

	return o
}

func (o *opt) SetSunZenith(sunZenith sunZenithEnum.SunZenith) option.Option {
	o.fajrZenith = sunZenith.FajrZenith()
	o.ishaZenith = sunZenith.IshaZenith().Angle
	o.ishaZenithType = sunZenith.IshaZenith().Type

	return o
}

func (o *opt) ValidateBySalat(salat salatEnum.Salat) error {
	if o.dateStart.IsZero() {
		return err.ErrDateMissing
	}

	if o.latitude.IsZero() {
		return err.ErrLatitudeMissing
	}

	if o.longitude.IsZero() {
		return err.ErrLongitudeMissing
	}

	if o.latitude.AngleType() != o.longitude.AngleType() {
		o.longitude = o.longitude.ToSpecificType(o.latitude.AngleType())
	}

	if o.timezoneLoc == nil {
		o.timezoneLoc = time.UTC
	}

	if o.fajrZenith.IsZero() && salat == salatEnum.Fajr {
		return err.ErrFajrZenithMissing
	}

	if o.ishaZenith.IsZero() && salat == salatEnum.Isha {
		return err.ErrIshaZenithMissing
	}

	if o.mazhab == 0 && salat == salatEnum.Asr {
		return err.ErrMazhabMissing
	}

	return nil
}

func (o *opt) CalculateSunPositions() (option.Option, error) {
	if len(o.sunPositions) != 0 {
		return o, nil
	}

	o.sunPositions = sunPositions.NewFromDateRange(o.dateStart, o.dateEnd, o.timezoneLoc, o.longitude)
	return o, nil
}

func (o *opt) CalculateFajrHighAltitude(declination angle.Angle) angle.Angle {
	return salatHighAltitude.CalcSalatHighAltitude(o.fajrZenith, o.latitude, declination, o.elevation)
}

func (o *opt) CalculateSunriseSunsetHighAltitude(declination angle.Angle) angle.Angle {
	return salatHighAltitude.CalcSalatHighAltitude(angle.NewDegreeFromFloat(consts.SunriseSunsetAngleFactor), o.latitude, declination, o.elevation)
}

func (o *opt) CalculateAsrAngle(declination angle.Angle) angle.Angle {
	return trig.Acos((trig.Sin(trig.Acot(o.mazhab.AsrShadowLength()+trig.Tan(o.latitude.Sub(declination).Abs()))) - (trig.Sin(o.latitude) * trig.Sin(declination))) / (trig.Cos(o.latitude) * trig.Cos(declination))).Div(15.)
}

func (o *opt) CalculateIshaHighAltitude(declination angle.Angle) (angle.Angle, sunZenithEnum.IshaZenithType) {
	if o.ishaZenithType == sunZenithEnum.Standard {
		return salatHighAltitude.CalcSalatHighAltitude(o.ishaZenith, o.latitude, declination, o.elevation), o.ishaZenithType
	}

	return o.ishaZenith, o.ishaZenithType
}

func (o *opt) RoundTime(t time.Time) time.Time {
	return o.roundingTimeOption.RoundTime(t)
}

func (o *opt) GetSunPositions() sunPositions.SunPositions {
	return o.sunPositions
}

func (o *opt) GetDateRange() (time.Time, time.Time) {
	return o.dateStart, o.dateEnd
}
