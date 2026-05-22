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
