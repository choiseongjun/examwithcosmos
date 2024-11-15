package main

import (
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"encoding/json"
	"fmt"
	db "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codeType "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Post struct {
	ID        int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	Title     string `protobuf:"bytes,2,opt,name=title,proto3" json:"title"`
	Content   string `protobuf:"bytes,3,opt,name=content,proto3" json:"content"`
	Author    string `protobuf:"bytes,4,opt,name=author,proto3" json:"author"`
	Timestamp int64  `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp"`
}

func (p *Post) Reset()         { *p = Post{} }
func (p *Post) String() string { return fmt.Sprintf("Post ID: %d, Title: %s", p.ID, p.Title) }
func (p *Post) ProtoMessage()  {}

type ForumKeeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.Codec
}

func (k ForumKeeper) setPost(ctx types.Context, post *Post) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("post:%d", post.ID))
	store.Set(key, k.cdc.MustMarshal(post))
}

func (k ForumKeeper) CreatePost(ctx types.Context, title, content, author string) {
	post := Post{
		ID:        time.Now().Unix(),
		Title:     title,
		Content:   content,
		Author:    author,
		Timestamp: ctx.BlockTime().Unix(),
	}

	// 상태 저장
	k.setPost(ctx, &post)

	// 게시글을 저장한 후 바로 가져오기
	storedPost, _ := k.GetPost(ctx, post.ID)

	// 상태 변경 시 이벤트 발생
	ctx.EventManager().EmitEvent(
		types.NewEvent(
			"post_created",
			types.NewAttribute("post_id", fmt.Sprintf("%d", storedPost.ID)),
			types.NewAttribute("title", storedPost.Title),
			types.NewAttribute("author", storedPost.Author),
		),
	)
}

//	func (k ForumKeeper) GetPost(ctx types.Context, id int64) (*Post, bool) {
//		store := ctx.KVStore(k.storeKey)
//		key := []byte(fmt.Sprintf("post:%d", id))
//		if !store.Has(key) {
//			return nil, false
//		}
//		var post Post
//		k.cdc.MustUnmarshal(store.Get(key), &post)
//		return &post, true
//	}
func (k ForumKeeper) GetPost(ctx types.Context, id int64) (*Post, bool) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("post:%d", id))
	if !store.Has(key) {
		return nil, false
	}
	var post Post
	k.cdc.MustUnmarshal(store.Get(key), &post)
	return &post, true
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var posts = make(map[int]map[string]string)
var postIDCounter = 1
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)
var mu sync.Mutex

// WebSocket 연결을 처리하는 함수
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// 클라이언트를 등록
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	// 연결이 닫힐 경우 클라이언트를 제거
	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
	}()

	//for {
	//	// 메시지를 받아서 처리할 수 있지만 현재는 필요하지 않음
	//	_, msg, err := conn.ReadMessage()
	//	if err != nil {
	//		fmt.Println("Error reading message:", err)
	//		break
	//	}
	//	broadcast <- string(msg)
	//
	//}
	for {
		// 메시지를 받아서 처리 (여기서 데이터 받기)
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		var post map[string]string
		if err := json.Unmarshal(msg, &post); err != nil {
			fmt.Println("Error unmarshalling message:", err)
			break
		}

		// 받은 데이터를 Keeper에 저장 (메모리 예시)
		postID := postIDCounter
		postIDCounter++
		posts[postID] = post

		// Keeper에 데이터가 저장되었는지 확인
		fmt.Printf("Post saved: ID %d, Title: %s, Author: %s\n", postID, post["title"], post["author"])

		// 저장된 데이터를 클라이언트에 broadcast
		broadcast <- string(msg)
	}
}

// 게시글이 생성되었을 때 WebSocket 클라이언트에게 메시지 전송
func broadcastMessage() {
	for {
		message := <-broadcast
		mu.Lock()
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				fmt.Println("Error sending message:", err)
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}
func (k ForumKeeper) GetPostHandler(w http.ResponseWriter, r *http.Request) {
	// URL에서 id 추출
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// 컨텍스트 생성
	ctx := k.createContext()

	// ID가 있으면 게시글 반환
	post, exists := k.GetPost(ctx, postID)
	if !exists {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// 게시글을 JSON 형태로 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// ForumKeeper에 컨텍스트 생성 메서드 추가
func (k ForumKeeper) createContext() sdk.Context {
	// MultiStore 생성
	db := db.NewMemDB()
	logger := log.NewNopLogger()
	metricGatherer := metrics.NewNoOpMetrics()
	//ms := store.NewCommitMultiStore(db)
	ms := store.NewCommitMultiStore(db, logger, metricGatherer)
	// storeKey를 MultiStore에 추가
	ms.MountStoreWithDB(k.storeKey, storetypes.StoreTypeIAVL, db)
	ms.LoadLatestVersion()

	// 블록 헤더 및 로거 설정
	// abci.Header를 사용하여 블록 헤더 생성
	header := types.Header{
		ChainID: "mychain",
	}
	// Context 생성
	ctx := sdk.NewContext(ms, header, false, logger)
	return ctx
}

//	func (k ForumKeeper) GetPostHandler(w http.ResponseWriter, r *http.Request) {
//		// URL에서 게시물 ID 파라미터 가져오기
//		//id := r.URL.Query().Get("id")
//		//if id == "" {
//		//	http.Error(w, "Post ID is required", http.StatusBadRequest)
//		//	return
//		//}
//		vars := mux.Vars(r)
//		id := vars["id"]
//		// 게시물 ID를 int64로 변환
//		postID, err := strconv.ParseInt(id, 10, 64)
//		if err != nil {
//			http.Error(w, "Invalid post ID", http.StatusBadRequest)
//			return
//		}
//
//		// 게시물 데이터 가져오기
//		post, found := k.GetPost(types.Context{}, postID)
//		if !found {
//			http.Error(w, "Post not found", http.StatusNotFound)
//			return
//		}
//
//		// 게시물 데이터를 JSON으로 응답
//		w.Header().Set("Content-Type", "application/json")
//		if err := json.NewEncoder(w).Encode(post); err != nil {
//			http.Error(w, "Failed to encode post data", http.StatusInternalServerError)
//		}
//	}
func main() {
	interfaceRegistry := codeType.NewInterfaceRegistry()
	storeKey := storetypes.NewKVStoreKey("forum")
	cdc := codec.NewProtoCodec(interfaceRegistry)
	forumKeeper := ForumKeeper{
		storeKey: storeKey,
		cdc:      cdc,
		//	storeKey storetypes.StoreKey
		//	cdc      codec.Codec
	}
	r := mux.NewRouter()

	http.HandleFunc("/ws", handleWebSocket)
	//http.HandleFunc("/posts", forumKeeper.GetPostHandler)
	http.HandleFunc("/posts", forumKeeper.GetPostHandler)

	//r.HandleFunc("/posts/{id:[0-9]+}", forumKeeper.GetPostHandler).Methods("GET")
	http.Handle("/", r) // mux 라우터로 핸들러 등록

	go broadcastMessage()

	fmt.Println("WebSocket server started on :8080")
	http.ListenAndServe(":8080", nil)
}
