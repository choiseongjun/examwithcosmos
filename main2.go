package main

import (
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"encoding/json"
	"fmt"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	db "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codeType "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
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

var logger log.Logger

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

	fmt.Println(post)
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

/*memdb 생성*/
//func createContext(k ForumKeeper) sdk.Context {
//	// 메모리 데이터베이스 생성
//	db := db.NewMemDB()
//	logger := log.NewNopLogger()
//	metricGatherer := metrics.NewNoOpMetrics()
//
//	// MultiStore 생성
//	ms := store.NewCommitMultiStore(db, logger, metricGatherer)
//
//	// Forum 스토어 키 등록
//	ms.MountStoreWithDB(k.storeKey, storetypes.StoreTypeIAVL, nil)
//
//	// MultiStore 초기화
//	if err := ms.LoadLatestVersion(); err != nil {
//		panic(fmt.Sprintf("Failed to load latest version: %v", err))
//	}
//
//	// Context 생성
//	header := cmtproto.Header{
//		ChainID: "mychain",
//		Height:  1,
//		Time:    time.Now(),
//	}
//
//	ctx := sdk.NewContext(ms, header, false, logger)
//	return ctx
//}

func init() {
	logger = initLogger()
}

func initLogger() log.Logger {
	// Zap 로거 설정
	zapConfig := zap.NewProductionConfig()
	zapConfig.OutputPaths = []string{"stdout"}
	//zapLogger, err := zapConfig.Build()
	//if err != nil {
	//	panic("Failed to initialize zap logger: " + err.Error())
	//}

	// Zap Logger를 io.Writer로 Wrapping
	writer := zapcore.AddSync(os.Stdout)

	// Cosmos SDK 호환 로거 생성
	return log.NewLogger(writer)
}
func createContext(k ForumKeeper) sdk.Context {
	// LevelDB 데이터베이스 생성
	logger := initLogger()

	database, err := db.NewGoLevelDB("forumdb", "./data", nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create LevelDB: %v", err))
	}

	// 로거 및 MultiStore 초기화
	//logWriter := zap.NewStdLog(logger)
	//zap.NewZapSugarLogger()
	//logWriter := zap.NewStdLog(logger)
	metricGatherer := metrics.NewNoOpMetrics()
	ms := store.NewCommitMultiStore(database, logger, metricGatherer)

	// Forum 스토어 키 등록
	ms.MountStoreWithDB(k.storeKey, storetypes.StoreTypeIAVL, nil)

	// MultiStore 초기화
	if err := ms.LoadLatestVersion(); err != nil {
		panic(fmt.Sprintf("Failed to load latest version: %v", err))
	}

	// Context 생성
	header := cmtproto.Header{
		ChainID: "mychain",
		Height:  1,
		Time:    time.Now(),
	}

	ctx := sdk.NewContext(ms, header, false, logger)
	return ctx
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// ForumKeeper 초기화
	forumKeeper := ForumKeeper{
		storeKey: storetypes.NewKVStoreKey("forum"),
		cdc:      codec.NewProtoCodec(codeType.NewInterfaceRegistry()),
	}
	ctx := createContext(forumKeeper) // KVStore를 사용하는 Context 생성

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트의 연결 종료 감지
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Printf("Unexpected WebSocket closure: %v\n", err)
			}
			break
		}

		var request map[string]string
		if err := json.Unmarshal(msg, &request); err != nil {
			fmt.Println("Error unmarshalling message:", err)
			break
		}

		action := request["action"]
		switch action {
		case "insert":
			// 게시글 저장
			title := request["title"]
			content := request["content"]
			author := request["author"]

			post := Post{
				ID:        time.Now().Unix(),
				Title:     title,
				Content:   content,
				Author:    author,
				Timestamp: time.Now().Unix(),
			}
			//logger.Info("New post created", "post", string(post))
			fmt.Println(post.ID)
			// KVStore에 저장
			forumKeeper.setPost(ctx, &post)

			response := map[string]string{
				"status":  "success",
				"action":  "insert",
				"id":      fmt.Sprintf("%d", post.ID),
				"title":   title,
				"content": content,
				"author":  author,
			}
			conn.WriteJSON(response)

		case "retrieve":
			// 게시글 조회
			idStr := request["id"]
			postID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				conn.WriteJSON(map[string]string{"error": "Invalid post ID"})
				continue
			}

			post, found := forumKeeper.GetPost(ctx, postID)
			if !found {
				conn.WriteJSON(map[string]string{"error": "Post not found"})
				continue
			}

			response := map[string]interface{}{
				"status":  "success",
				"action":  "retrieve",
				"id":      post.ID,
				"title":   post.Title,
				"content": post.Content,
				"author":  post.Author,
			}
			conn.WriteJSON(response)

		default:
			conn.WriteJSON(map[string]string{"error": "Invalid action"})
		}
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
	//header := types.Header{
	//	ChainID: "mychain",
	//}
	header := cmtproto.Header{
		ChainID: "mychain",
		Height:  1,
		Time:    time.Now(),
	}
	// Context 생성
	ctx := sdk.NewContext(ms, header, false, logger)
	return ctx
}
func main() {
	storeKey := storetypes.NewKVStoreKey("forum")
	interfaceRegistry := codeType.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	forumKeeper := ForumKeeper{
		storeKey: storeKey,
		cdc:      cdc,
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
