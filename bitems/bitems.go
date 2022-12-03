package bitems

import "github.com/golang/glog"

const (
	BINT = iota
	BSTRING
	BLIST
	BDICTIONARY
)

// type BItemInterface interface {
// 	GetType() (b_type int)
// 	GetValue() (b_val interface{})
// }

type BITEM struct {
	b_type int
	b_val  interface{}
}

func (item *BITEM) GetType() (btype int) {
	if item == nil {
		glog.Fatalln("BITEM nullptr err")
	}
	return btype
}

func (item *BITEM) SetType(btype int) (err error) {
	if item == nil {
		glog.Fatalln("BITEM nullptr err")
		return err
	}
	if btype != BINT && btype != BSTRING && btype != BLIST && btype != BDICTIONARY {
		glog.Fatalln("BITEM type err")
		return err
	}
	item.b_type = btype
	return err
}

// func (item *BITEM) GetType() (btype int) {
// 	if item == nil {
// 		glog.Fatalln("BITEM nullptr err")
// 	}
// 	return btype
// }

func (item *BITEM) SetValue(val interface{}) (err error) {
	if item == nil || val == nil {
		glog.Fatalln("nullptr, Check please")
		return err
	}
	item.b_val = val
	return err
}
