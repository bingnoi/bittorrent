package main

import (
	"log"
	"os"
	"github.com/bingnoi/bittorrent/torrentfile"
)

func main() {

	//这部分是处理了空值的情况，防止出现用户不提供完整值的情况
	inTorrentPath := ""
	outFilePath := ""

	if os.Args[1] == "" { 
		log.Println("input file cannot empty!")
		return
	} else{
		inTorrentPath = os.Args[1]
	}
	 
	if len(os.Args) == 2 {
		log.Println("output file cannot empty! Set Default name already")
		outFilePath = "default.iso"
	} else {
		outFilePath = os.Args[2]
	}

	//打开并解析torrent文件
	tf, err := torrentfile.Open(inTorrentPath)
	if err != nil {
		log.Fatal(err)
	}

	//下载对应的pieces并完成拼接
	err = tf.DownloadToFile(outFilePath)
	if err != nil {
		log.Fatal(err)
	}
}
