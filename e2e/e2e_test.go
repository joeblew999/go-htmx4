package e2e

import "testing"

// TestE2E runs the browser checks side by side (each in its own Chrome; -parallel, default GOMAXPROCS), then the
// write limit alone and last: it uses up this client's write budget, which the board checks need. Run one check with
// go test -run 'TestE2E/parallel/Pages'.
func TestE2E(t *testing.T) {
	t.Run("parallel", func(t *testing.T) {
		for _, c := range []struct {
			name string
			run  func(*testing.T)
		}{
			{"BoardTwoBrowsers", testBoardTwoBrowsers},
			{"BoostBoard", testBoostBoard},
			{"BareClosePresence", testBareClosePresence},
			{"ViewerTimeZone", testViewerTimeZone},
			{"NoOverflow", testNoOverflow},
			{"ArabicDigitsAndMirroring", testArabicDigitsAndMirroring},
			{"LocaleSwitchKeepsPreferences", testLocaleSwitchKeepsPreferences},
			{"MixedLocalePush", testMixedLocalePush},
			{"IntlKinds", testIntlKinds},
			{"Locales", testLocales},
			{"LanguageOffer", testLanguageOffer},
			{"Pages", testPages},
			{"Components", testComponents},
			{"LiveVideo", testLiveVideo},
			{"Map", testMap},
		} {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				c.run(t)
			})
		}
	})
	t.Run("WriteLimit", testWriteLimit)
}
