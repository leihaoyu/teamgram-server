package service

import (
	"context"
	"time"

	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/core"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// warmupStartDelay is the time to wait after service start before fetching featured sets,
	// giving upstream services (idgen, media, dfs) time to become fully ready.
	warmupStartDelay = 5 * time.Second

	// warmupInterSetDelay is the pause between consecutive featured-set downloads
	// to spread the CPU/memory load across time.
	warmupInterSetDelay = 10 * time.Second
)

type Service struct {
	svcCtx *svc.ServiceContext
}

func New(ctx *svc.ServiceContext) *Service {
	s := &Service{svcCtx: ctx}
	go s.warmupFeaturedSetsInBackground()
	return s
}

// warmupFeaturedSetsInBackground pre-fetches each configured FeaturedStickerSet into the DB cache
// at startup, so the first user request does not trigger a large synchronous download spike.
// It waits warmupStartDelay for upstream services to be ready, then fetches one set at a time
// with warmupInterSetDelay between sets to keep CPU and memory usage smooth.
func (s *Service) warmupFeaturedSetsInBackground() {
	names := s.svcCtx.Config.FeaturedStickerSets
	log := logx.WithContext(context.Background())

	if len(names) == 0 {
		log.Infof("warmupFeaturedSets - no FeaturedStickerSets configured, skipping warmup")
		return
	}

	log.Infof("warmupFeaturedSets - scheduled for %d sets: %v", len(names), names)

	// Wait for upstream services to be fully ready before making RPC calls.
	time.Sleep(warmupStartDelay)

	for i, name := range names {
		if i > 0 {
			// Spread downloads over time to avoid CPU/memory spikes.
			time.Sleep(warmupInterSetDelay)
		}

		bgCtx := context.Background()

		// Skip if already cached to avoid re-downloading on restart.
		setDO, err := s.svcCtx.Dao.StickerSetsDAO.SelectByShortName(bgCtx, name)
		if err != nil {
			log.Errorf("warmupFeaturedSets - SelectByShortName(%s) error: %v", name, err)
			continue
		}
		if setDO != nil {
			log.Infof("warmupFeaturedSets - %s already cached, skipping", name)
			continue
		}

		log.Infof("warmupFeaturedSets - fetching %s (%d/%d)", name, i+1, len(names))
		c := core.New(bgCtx, s.svcCtx)
		if _, err := c.FetchAndCacheStickerSet(name); err != nil {
			log.Errorf("warmupFeaturedSets - FetchAndCacheStickerSet(%s) error: %v", name, err)
		} else {
			log.Infof("warmupFeaturedSets - %s cached successfully", name)
		}
	}

	log.Infof("warmupFeaturedSets - warmup complete")
}
