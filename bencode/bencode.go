package bencode

import (
	"bufio"
	"bitems"

	"github.com/golang/glog"
)

func ReadNumber(reader *bufio.Reader) (value int) {
	isNegative := false
	for {
		if char, _ := reader.Peek(1); char[0] == ':' || char[0] == 'e' {
			// reach end of int when ':' 'e'
			break
		} else {
			ch, _ := reader.ReadByte()
			if ch == '-' {
				isNegative = true
				continue
			}
			value = value*10 + (int(ch) - 48) // byte2int
		}
	}
	if isNegative {
		value = -value
	}
	return value
}

func DecodeBString(reader *bufio.Reader) (b_string string, err error) {
	// get string length
	stringLength := ReadNumber(reader)
	glog.Infoln(stringLength)
	// read colon
	colon, err := reader.ReadByte()
	if colon != ':' || err != nil {
		return "err", err
	}
	// read string
	for i := 0; i < stringLength; i++ {
		ch, _ := reader.ReadByte()
		b_string += string(ch)
	}
	return b_string, err
}

func DecodeBInteger(reader *bufio.Reader) (b_integer int, err error) {
	flag, err := reader.ReadByte()
	if flag != 'i' || err != nil {
		glog.Fatalln("fail")
		return 0, err
	}
	// read int number
	b_integer = ReadNumber(reader)
	// read ending flag 'e'
	flag, err = reader.ReadByte()
	if flag != 'e' || err != nil {
		glog.Fatalln("fail")
		return 0, err
	}
	return b_integer, err
}

func RecursiveParse(reader *bufio.Reader) (*BITEM, error) {
	typeFlag, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}
	var retItem BITEM
	switch {
	case typeFlag[0] == 'i':
		// Parse Integer
		number, err := DecodeBInteger(reader)
		if err != nil {
			return nil, err
		}
		glog.Infoln("int number parsed:", number)
		retItem.SetType(BINT)
		retItem.SetValue(number)
	case typeFlag[0] > '0' && typeFlag[0] < '9':
		// Parse String
		str, err := DecodeBString(reader)
		if err != nil {
			return nil, err
		}
		glog.Infoln("string parsed:", str)
		retItem.SetType(BSTRING)
		retItem.SetValue(str)
	case typeFlag[0] == 'l':
		// Parse List
		// read list start flag 'l'
		reader.ReadByte()
		var curList []*BITEM
		for {
			// 'e' ends list
			flag, err := reader.Peek(1)
			if err != nil {
				return nil, err
			}
			if flag[0] == 'e' {
				// read list end flag 'e'
				reader.ReadByte()
				break
			}
			newElement, err := RecursiveParse(reader)
			if err != nil {
				return nil, err
			}
			curList = append(curList, newElement)
		}
		retItem.SetType(BLIST)
		retItem.SetValue(curList)
		for index, value := range curList {
			glog.Infoln("index:", index, "value", value)
		}
	case typeFlag[0] == 'd':
		// Parse Dictionary
		// read dictionary start flag 'd'
		reader.ReadByte()
		curDict := make(map[string]*BITEM)
		for {
			// 'e' ends dictionary
			flag, err := reader.Peek(1)
			if err != nil {
				return nil, err
			}
			if flag[0] == 'e' {
				// read list end flag 'e'
				reader.ReadByte()
				break
			}
			key, err := DecodeBString(reader)
			if err != nil {
				return nil, err
			}
			value, err := RecursiveParse(reader)
			if err != nil {
				return nil, err
			}
			if err != nil {
				return nil, err
			}
			curDict[key] = value
		}
		retItem.SetType(BDICTIONARY)
		retItem.SetValue(curDict)
	}
	return &retItem, err
}

// func Unmarshal()
