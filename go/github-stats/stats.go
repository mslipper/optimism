package github_stats

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/go-github/v43/github"
	"sync"
	"sync/atomic"
)

const (
	MaxPRs = 100
)

type dataBag struct {
	pr      *github.PullRequest
	reviews []*github.PullRequestReview
}

func CalculateStats(gh *github.Client, org, repo string) error {
	ctx := context.Background()

	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 25,
		},
	}
	var allPrs []*github.PullRequest
	for {
		prs, res, err := gh.PullRequests.List(ctx, org, repo, opts)
		if err != nil {
			return wrapErr(err, "error fetching pull requests")
		}

		allPrs = append(allPrs, prs...)
		if res.NextPage == 0 || len(allPrs) >= MaxPRs {
			break
		}
		opts.Page = res.NextPage
	}

	reviewCount := make(map[string]float64)
	sem := make(chan struct{}, 8)
	bagCh := make(chan *dataBag)
	var allDone sync.WaitGroup
	allDone.Add(1)

	var isErr int32
	var totalResolutionTime float64
	var totalDiffSizes float64
	var prCount float64

	go func() {
		var i int
		for bag := range bagCh {
			for _, review := range bag.reviews {
				reviewCount[*review.User.Login] += 1
			}

			totalDiffSizes += float64(bag.pr.GetAdditions() + bag.pr.GetDeletions())

			i++
			log.Info("processed PR", "current", i, "total", len(allPrs))
		}

		allDone.Done()
	}()

	for _, pr := range allPrs {
		sem <- struct{}{}

		if pr.ClosedAt != nil {
			totalResolutionTime += pr.GetClosedAt().Sub(pr.GetCreatedAt()).Seconds()
			prCount++
		}

		go func(pr *github.PullRequest) {
			defer func() { <-sem }()

			if atomic.LoadInt32(&isErr) == 1 {
				return
			}

			fullPr, _, err := gh.PullRequests.Get(ctx, org, repo, *pr.Number)
			if err != nil {
				log.Error("error fetching full PR attributes", "err", err)
				atomic.StoreInt32(&isErr, 1)
				return
			}

			reviews, _, err := gh.PullRequests.ListReviews(ctx, org, repo, *pr.Number, &github.ListOptions{
				PerPage: 25,
			})
			if err != nil {
				log.Error("error listing pull requests", "err", err)
				atomic.StoreInt32(&isErr, 1)
				return
			}

			bagCh <- &dataBag{
				pr:      fullPr,
				reviews: reviews,
			}
		}(pr)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	close(bagCh)
	allDone.Wait()

	if isErr == 1 {
		return errors.New("an error occurred fetching reviews, check the logs for the source")
	}

	PRMeanResolutionTimeSecs.Set(totalResolutionTime / prCount)
	PRMeanDiffSizeLines.Set(totalDiffSizes / prCount)
	for reviewer, count := range reviewCount {
		PRReviewsCount.WithLabelValues(reviewer).Set(count)
	}

	return nil
}

func wrapErr(err error, msg string) error {
	return fmt.Errorf("%s: %v", msg, err)
}
