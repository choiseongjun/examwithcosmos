package main

import (
	"cosmossdk.io/store/metrics"
	"fmt"
	db "github.com/cosmos/cosmos-db"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Post 구조체 정의 및 Protobuf 인터페이스 구현
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

func (k ForumKeeper) setPost(ctx sdk.Context, post *Post) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("post:%d", post.ID))
	store.Set(key, k.cdc.MustMarshal(post))
}

func (k ForumKeeper) CreatePost(ctx sdk.Context, title, content, author string) {
	post := Post{
		ID:        1,
		Title:     title,
		Content:   content,
		Author:    author,
		Timestamp: ctx.BlockTime().Unix(),
	}
	k.setPost(ctx, &post)
}

func (k ForumKeeper) GetPost(ctx sdk.Context, id int64) (*Post, bool) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("post:%d", id))
	if !store.Has(key) {
		return nil, false
	}
	var post Post
	k.cdc.MustUnmarshal(store.Get(key), &post)
	return &post, true
}

func main() {
	// Cosmos SDK 설정
	storeKey := storetypes.NewKVStoreKey("forum")
	interfaceRegistry := types.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// ForumKeeper 인스턴스 생성
	forumKeeper := ForumKeeper{
		storeKey: storeKey,
		cdc:      cdc,
	}
	metricGatherer := metrics.NewNoOpMetrics() // 메트릭 수집기 생성
	logger := log.NewNopLogger()
	// MemDB 데이터베이스 초기화
	db := db.NewMemDB()
	cms := store.NewCommitMultiStore(db, logger, metricGatherer)
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		panic(fmt.Sprintf("failed to load latest version: %v", err))
	}

	// 블록 헤더 생성 및 컨텍스트 생성
	header := tmproto.Header{
		Time: time.Now(),
	}

	ctx := sdk.NewContext(cms, header, false, logger)

	// 게시글 생성 및 조회 테스트
	forumKeeper.CreatePost(ctx, "Hello Cosmos", "This is a test post", "user1")
	post, found := forumKeeper.GetPost(ctx, 1)
	if found {
		fmt.Printf("Post found: %+v\n", post)
	} else {
		fmt.Println("Post not found")
	}
}
