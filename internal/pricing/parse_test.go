package pricing

import "testing"

const samplePriceListGp3 = `{
  "product": {
    "productFamily": "Storage",
    "attributes": {
      "volumeApiName": "gp3"
    }
  },
  "terms": {
    "OnDemand": {
      "ABC.JRTCKXETXF": {
        "priceDimensions": {
          "ABC.JRTCKXETXF.6YS6EN2CT7": {
            "unit": "GB-Mo",
            "pricePerUnit": {"USD": "0.08"}
          }
        }
      }
    }
  }
}`

const samplePriceListGp2 = `{
  "product": {
    "productFamily": "Storage",
    "attributes": {"volumeApiName": "gp2"}
  },
  "terms": {
    "OnDemand": {
      "X.Y": {
        "priceDimensions": {
          "X.Y.Z": {
            "unit": "GB-Mo",
            "pricePerUnit": {"USD": "0.10"}
          }
        }
      }
    }
  }
}`

const samplePriceListInstance = `{
  "product": {
    "productFamily": "Compute Instance",
    "attributes": {"instanceType": "t3.micro"}
  },
  "terms": {}
}`

const samplePriceListEmptyVolType = `{
  "product": {
    "productFamily": "Storage",
    "attributes": {"volumeApiName": ""}
  },
  "terms": {}
}`

const samplePriceListIO2 = `{
  "product": {
    "productFamily": "Storage",
    "attributes": {"volumeApiName": "io2"}
  },
  "terms": {
    "OnDemand": {
      "X64.JRTCKXETXF": {
        "priceDimensions": {
          "X64.JRTCKXETXF.6YS6EN2CT7": {
            "unit": "GB-month",
            "pricePerUnit": {"USD": "0.142"}
          }
        }
      }
    }
  }
}`

const samplePriceListIOPSOnly = `{
  "product": {
    "productFamily": "Storage",
    "attributes": {"volumeApiName": "io2"}
  },
  "terms": {
    "OnDemand": {
      "X.Y": {
        "priceDimensions": {
          "X.Y.Z": {
            "unit": "IOPS-Mo",
            "pricePerUnit": {"USD": "0.065"}
          }
        }
      }
    }
  }
}`

func TestParsePriceListItem_GP3(t *testing.T) {
	vt, rate, ok, err := parsePriceListItem(samplePriceListGp3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok || vt != "gp3" || rate != 0.08 {
		t.Errorf("got (%q, %f, %v), want (gp3, 0.08, true)", vt, rate, ok)
	}
}

func TestParsePriceListItem_IO2_GBmonth(t *testing.T) {
	// Regression: io2 PriceList items use "GB-month" instead of "GB-Mo",
	// which the original parser silently dropped. Real example pulled from
	// AWS Pricing API (ap-northeast-1).
	vt, rate, ok, err := parsePriceListItem(samplePriceListIO2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok || vt != "io2" || rate != 0.142 {
		t.Errorf("got (%q, %f, %v), want (io2, 0.142, true)", vt, rate, ok)
	}
}

func TestIsPerGBMonthlyUnit(t *testing.T) {
	cases := map[string]bool{
		"GB-Mo":    true,
		"GB-month": true,
		"GB-Month": true,
		"gb-mo":    true,
		"  GB-Mo ": true,
		"IOPS-Mo":  false,
		"":         false,
		"GiB-Mo":   false,
		"hours":    false,
	}
	for in, want := range cases {
		if got := isPerGBMonthlyUnit(in); got != want {
			t.Errorf("isPerGBMonthlyUnit(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParsePriceListItem_SkipsNonStorage(t *testing.T) {
	_, _, ok, err := parsePriceListItem(samplePriceListInstance)
	if err != nil || ok {
		t.Errorf("expected skip with no err, got ok=%v err=%v", ok, err)
	}
}

func TestParsePriceListItem_SkipsEmptyVolType(t *testing.T) {
	_, _, ok, err := parsePriceListItem(samplePriceListEmptyVolType)
	if err != nil || ok {
		t.Errorf("expected skip, got ok=%v err=%v", ok, err)
	}
}

func TestParsePriceListItem_SkipsIOPSDimension(t *testing.T) {
	_, _, ok, err := parsePriceListItem(samplePriceListIOPSOnly)
	if err != nil || ok {
		t.Errorf("expected skip for IOPS-Mo dimension, got ok=%v err=%v", ok, err)
	}
}

func TestParsePriceListItem_InvalidJSON(t *testing.T) {
	_, _, _, err := parsePriceListItem("not json")
	if err == nil {
		t.Fatal("expected error")
	}
}
