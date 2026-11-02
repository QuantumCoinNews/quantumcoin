package consensus

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNFTMonthlyProgramHasExactlyTwelveActivationDerivedPeriodsV1(t *testing.T) {
	activation := time.Date(2031, time.November, 17, 12, 0, 0, 0, time.UTC)
	periods, err := MonthlyNFTAwardPeriodsFromGameActivationV1(activation)
	if err != nil {
		t.Fatal(err)
	}
	if len(periods) != 12 || periods[0] != "2031-12" || periods[11] != "2032-11" {
		t.Fatalf("unexpected periods: %v", periods)
	}
	p, err := NewNFTMonthlyProgramPolicyFromGameActivationV1("test-only-monthly", "telegram-monthly", activation)
	if err != nil {
		t.Fatal(err)
	}
	for i, period := range periods {
		serial, err := p.SerialForAwardPeriodV1(period)
		if err != nil || serial != uint32(i+1) {
			t.Fatalf("period %s serial=%d err=%v", period, serial, err)
		}
	}
}

func TestNFTMonthlyProgramRejectsOutsideWindowAndMalformedScopeV1(t *testing.T) {
	p := NFTMonthlyProgramPolicyV1{NetworkID: "test-only-monthly", CampaignID: "telegram-monthly", FirstAwardPeriod: "2031-12"}
	for _, period := range []string{"2031-11", "2032-12", "2033-01", "", "2032-13", "2032-1"} {
		if _, err := p.SerialForAwardPeriodV1(period); err == nil {
			t.Fatalf("period %q accepted", period)
		}
	}
	for _, invalid := range []NFTMonthlyProgramPolicyV1{
		{}, {NetworkID: "bad network", CampaignID: p.CampaignID, FirstAwardPeriod: p.FirstAwardPeriod},
		{NetworkID: p.NetworkID, CampaignID: "bad campaign", FirstAwardPeriod: p.FirstAwardPeriod},
		{NetworkID: p.NetworkID, CampaignID: p.CampaignID, FirstAwardPeriod: "0000-01"},
		{NetworkID: p.NetworkID, CampaignID: p.CampaignID, FirstAwardPeriod: "9999-12"},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("invalid policy accepted: %+v", invalid)
		}
	}
}

func TestNFTMonthlyProgramUsesGameActivationAndUTCWithoutFixedDateV1(t *testing.T) {
	a := time.Date(2035, time.January, 1, 1, 0, 0, 0, time.FixedZone("test-plus-three", 3*60*60))
	x, err := MonthlyNFTAwardPeriodsFromGameActivationV1(a)
	if err != nil {
		t.Fatal(err)
	}
	y, err := MonthlyNFTAwardPeriodsFromGameActivationV1(a.UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) || x[0] != "2035-01" {
		t.Fatalf("timezone-dependent periods: %v %v", x, y)
	}
	later, err := MonthlyNFTAwardPeriodsFromGameActivationV1(a.AddDate(1, 0, 0))
	if err != nil || later[0] == x[0] {
		t.Fatal("activation date was ignored")
	}
	if _, err := MonthlyNFTAwardPeriodsFromGameActivationV1(time.Time{}); !errors.Is(err, ErrMonthlyNFTGameActivationRequiredV1) {
		t.Fatalf("missing activation: %v", err)
	}
}
