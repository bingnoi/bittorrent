package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/bingnoi/bittorrent/client"
	"github.com/bingnoi/bittorrent/message"
	"github.com/bingnoi/bittorrent/peers"
)

const MaxBlockSize = 16384

const MaxBacklog = 5

type Torrent struct {
	Peers       []peers.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type filePiece struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.MsgUnchoke:
		state.client.Choked = false
	case message.MsgChoke:
		state.client.Choked = true
	case message.MsgHave:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)
	case message.MsgPiece:
		n, err := message.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, pw *filePiece) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	//设置最大超时时长
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{})

	for state.downloaded < pw.length {
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize

				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

//通过哈希算法检查完整性
func checkIntegrity(pw *filePiece, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("Index %d failed, Check please", pw.index)
	}
	return nil
}


//这个地方是为了实现下载pieces，分为1、建立handshake，发送unchoke 2、获取pieces
func (torr *Torrent) startDownloadWorker(peer peers.Peer, downloadQueue chan *filePiece, results chan *pieceResult) {

	//1、新建client，进行handshake
	c, err := client.New(peer, torr.PeerID, torr.InfoHash)
	if err != nil {
		log.Printf("Connecting with %s .... HandShake Fail\n", peer.IP)
		return
	}
	defer c.Conn.Close()
	log.Printf("Connecting with %s .... HandShake OK\n", peer.IP)

	//发送相关信息
	c.SendUnchoke()
	c.SendInterested()


	//开启下载队列
	for pw := range downloadQueue {
		if !c.Bitfield.HasPiece(pw.index) {
			downloadQueue <- pw
			continue
		}

		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			log.Println("Bye", err)
			downloadQueue <- pw
			return
		}

		err = checkIntegrity(pw, buf)
		if err != nil {
			log.Printf("Piece #%d not right, Check please\n", pw.index)
			downloadQueue <- pw
			continue
		}

		c.SendHave(pw.index)
		results <- &pieceResult{pw.index, buf}
	}
}

//计算边界，用于处理完整性的
func (torr *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * torr.PieceLength
	end = begin + torr.PieceLength
	if end > torr.Length {
		end = torr.Length
	}
	return begin, end
}

func (torr *Torrent) calculatePieceSize(index int) int {
	begin, end := torr.calculateBoundsForPiece(index)
	return end - begin
}

//下载pieces
func (torr *Torrent) Download() ([]byte, error) {
	log.Println("Now, We are downloading file : ", torr.Name)

	//生成队列
	downloadQueue := make(chan *filePiece, len(torr.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range torr.PieceHashes {
		length := torr.calculatePieceSize(index)
		downloadQueue <- &filePiece{index, hash, length}
	}

	//生成peers对象,并下载
	for _, peer := range torr.Peers {
		go torr.startDownloadWorker(peer, downloadQueue, results)
	}

	buf := make([]byte, torr.Length)
	donePieces := 0

	//对于每个piece
	for donePieces < len(torr.PieceHashes) {
		res := <-results
		begin, end := torr.calculateBoundsForPiece(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		percent := float64(donePieces) / float64(len(torr.PieceHashes)) * 100
		numWorkers := runtime.NumGoroutine() - 1
		log.Printf("(We have gone through (%0.2f%%)), #%d --(piece)--> #%d", percent, numWorkers, res.index)
	}
	close(downloadQueue)

	return buf, nil
}
