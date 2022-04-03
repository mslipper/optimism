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
	sem := make(chan struct{}, 4)
	revsCh := make(chan []*github.PullRequestReview)
	var allDone sync.WaitGroup
	allDone.Add(1)

	var isErr int32
	var total float64
	var count float64

	go func() {
		var i int
		for reviews := range revsCh {
			for _, review := range reviews {
				reviewCount[*review.User.Login] += 1
			}
			i++
			log.Info("processed PR", "current", i, "total", len(allPrs))
		}

		allDone.Done()
	}()

	for _, pr := range allPrs {
		sem <- struct{}{}

		if pr.ClosedAt != nil {
			total += pr.GetClosedAt().Sub(pr.GetCreatedAt()).Seconds()
			count++
		}

		go func(pr *github.PullRequest) {
			defer func() { <- sem }()

			reviews, _, err := gh.PullRequests.ListReviews(ctx, org, repo, *pr.Number, &github.ListOptions{
				PerPage: 25,
			})
			if err != nil {
				log.Error("error listing pull requests", "err", err)
				atomic.StoreInt32(&isErr, 1)
				return
			}

			revsCh <- reviews
		}(pr)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	close(revsCh)
	allDone.Wait()

	if isErr == 1 {
		return errors.New("an error occurred fetching reviews, check the logs for the source")
	}

	PRMeanResolutionTimeSecs.Set(total / count)
	for reviewer, count := range reviewCount {
		PRReviewsCount.WithLabelValues(reviewer).Set(count)
	}

	return nil
}

func wrapErr(err error, msg string) error {
	return fmt.Errorf("%s: %v", msg, err)
}
