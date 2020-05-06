package main

import "gopkg.in/src-d/go-git.v4"

func gitVerifyHeadCommitSignature(repo *git.Repository, armoredKeyRing string) error {
	headRef, err := repo.Head()
	if err != nil {
		return err
	}
	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}
	_, err = commit.Verify(armoredKeyRing)
	if err != nil {
		return err
	}
	return nil
}

func gitCloneOrOpen(cfg *config) (*git.Repository, error) {
	repo, err := git.PlainClone(cfg.OutputDir, false, &git.CloneOptions{
		URL:           cfg.Repository,
		SingleBranch:  true,
		Depth:         cfg.CloneDepth,
	})
	if err != nil {
		if err != git.ErrRepositoryAlreadyExists {
			return nil, err
		}
		repo, err = git.PlainOpen(cfg.OutputDir)
	}
	return repo, err
}

func gitPullChanges(repo *git.Repository, cfg *config) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	return wt.Pull(&git.PullOptions{
		SingleBranch:  true,
		Depth:         cfg.CloneDepth,
		Force:         true,
	})
}
