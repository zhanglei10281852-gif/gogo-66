package locate

import (
	"testing"

	"CableMend/internal/config"
	"CableMend/internal/model"
)

// coverageIndex is a plain 100 km single-segment route with a 1.05 slack factor,
// so a cable distance of 21.0 km from end A is route KP 20.0.
func coverageIndex() *model.SystemIndex {
	sys := model.CableSystem{
		ID:          "SYS-W",
		Name:        "Window Link",
		SlackFactor: 1.05,
		LandingStations: []model.LandingStation{
			{ID: "LS-A", KP: 0, Jurisdiction: "JA"},
			{ID: "LS-B", KP: 100, Jurisdiction: "JB"},
		},
		Segments: []model.Segment{{
			ID: "SEG-1", StartKP: 0, EndKP: 100, CableType: "da", SlackFactor: 1.05, LossDbPerKm: 0.2,
			BurialProfile: []model.BurialSpan{
				{StartKP: 0, EndKP: 100, DepthM: 1800, BurialDepthM: 0, Seabed: "clay"},
			},
		}},
	}
	return model.NewSystemIndex(sys, 1.03)
}

// coverageEvidence is an optical measurement with an explicit one-sigma
// uncertainty so the combined uncertainty does not depend on the defaults.
func coverageEvidence(id string, end model.End, cableKm, sigmaKm float64) model.Evidence {
	cable := cableKm
	sigma := sigmaKm
	return model.Evidence{
		ID:              id,
		SystemID:        "SYS-W",
		FaultID:         "F-9",
		ObservedAt:      model.MustParseUTC("2026-03-10T00:00:00Z"),
		End:             end,
		Method:          model.MethodOTDR,
		Instrument:      "OTDR-" + id,
		CableDistanceKm: &cable,
		UncertaintyKm:   &sigma,
	}
}

// TestLocalizeWindowBracketsEveryMeasurement pins the reported uncertainty window
// against the measurements it is derived from: the window must contain every
// per-record route estimate and both per-end estimates, while agreeing evidence
// must still produce a narrow window.
func TestLocalizeWindowBracketsEveryMeasurement(t *testing.T) {
	cfg := config.Default()
	ix := coverageIndex()

	// End A puts the fault at KP 20.0 and end B at KP 25.0.
	disagreeing := []model.Evidence{
		coverageEvidence("EV-A", model.EndA, 21.0, 0.4),
		coverageEvidence("EV-B", model.EndB, 78.75, 0.4),
	}
	res, err := Localize(cfg, ix, disagreeing, "F-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Estimates) != 2 {
		t.Fatalf("estimates = %d, want one per record", len(res.Estimates))
	}
	const eps = 1e-6
	for _, est := range res.Estimates {
		if est.RouteKP < res.WindowLowKP-eps || est.RouteKP > res.WindowHighKP+eps {
			t.Fatalf("estimate %s at KP %v is outside the reported window %v..%v",
				est.EvidenceID, est.RouteKP, res.WindowLowKP, res.WindowHighKP)
		}
	}
	if res.EndEstimateAKP < res.WindowLowKP-eps || res.EndEstimateAKP > res.WindowHighKP+eps {
		t.Fatalf("end A estimate %v is outside the window %v..%v",
			res.EndEstimateAKP, res.WindowLowKP, res.WindowHighKP)
	}
	if res.EndEstimateBKP < res.WindowLowKP-eps || res.EndEstimateBKP > res.WindowHighKP+eps {
		t.Fatalf("end B estimate %v is outside the window %v..%v",
			res.EndEstimateBKP, res.WindowLowKP, res.WindowHighKP)
	}
	if res.WindowWidthKm < 4.9 {
		t.Fatalf("window width = %v km, too narrow for estimates 5 km apart", res.WindowWidthKm)
	}
	if res.UncertaintyKm < 2.4 {
		t.Fatalf("reported uncertainty = %v km, want at least the worst deviation", res.UncertaintyKm)
	}

	// Agreeing evidence must not be widened: both records place the fault at KP 20.
	agreeing := []model.Evidence{
		coverageEvidence("EV-C", model.EndA, 21.0, 0.4),
		coverageEvidence("EV-D", model.EndB, 84.0, 0.4),
	}
	tight, err := Localize(cfg, ix, agreeing, "F-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tight.WindowWidthKm > 1.5 {
		t.Fatalf("window width = %v km, agreeing measurements must stay tight", tight.WindowWidthKm)
	}
	if tight.WindowWidthKm < cfg.Localization.MinWindowKm {
		t.Fatalf("window width = %v km, below the configured minimum", tight.WindowWidthKm)
	}
}
