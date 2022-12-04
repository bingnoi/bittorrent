/*This file is the first function file, dealing with the open of a torrent file
and dispatch the function to various module*/

package torrentfile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/bingnoi/bittorrent/p2p"
	"github.com/bingnoi/bittorrent/peers"
	"github.com/jackpal/bencode-go"
)

const Port uint16 = 6881


// define a decoded torrent file

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

//define a bencode
type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

type bencodeTrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (torr*TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	//读取PeerId,并进行处理
	peers, err := torr.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    torr.InfoHash,
		PieceHashes: torr.PieceHashes,
		PieceLength: torr.PieceLength,
		Length:      torr.Length,
		Name:        torr.Name,
	}

	//开始下载
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	//创建文件入口
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}

	log.Println("Download Successfully! Wish you have a good day!")
	return nil
}

func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)

	//如果解码失败
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	fmt.Println(path)

	//如果解码成功
	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)

	if err != nil {
		return TorrentFile{}, err
	}

	return bto.toTorrentFile()
}

func (info *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *info)
	if err != nil {
		return [20]byte{}, err
	}
	hashsum := sha1.Sum(buf.Bytes())
	return hashsum, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // 哈希的长度
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Error! length %d not right", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	torr:= TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	log.Println("Announce...OK")
	log.Println("InfoHash...OK")
	log.Println("PieceHashes...OK")  
	log.Println("PieceLength...OK")  
	log.Println("Length...OK")       
	log.Println("Name...OK")     
	log.Println("Parse Successfully:)")    
	return torr , nil
}

func (torr *TorrentFile) buildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(torr.Announce)
	if err != nil {
		return "", err
	}
	params := url.Values{
		"info_hash":  []string{string(torr.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(Port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(torr.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}

func (torr*TorrentFile) requestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
	//构建TrackerUrl
	url, err := torr.buildTrackerURL(peerID, port)

	if err != nil {
		return nil, err
	}

	//设置超时时间与事件
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	trackerResp := bencodeTrackerResp{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	return peers.Unmarshal([]byte(trackerResp.Peers))
}
