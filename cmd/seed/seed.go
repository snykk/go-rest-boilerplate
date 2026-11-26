package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
)

// helloWorld is the slug of the published article the reader comments on.
const helloWorld = "hello-world"

// Accounts are the credentials of the two accounts Seed creates: an editor, who holds the roles reader and
// editor, and a reader.
type Accounts struct {
	EditorEmail, EditorPassword, ReaderEmail, ReaderPassword string
}

// seeder holds what the steps of Seed share.
type seeder struct {
	log   *slog.Logger
	users *identity.Module
	api   client
}

// Seed fills the database with sample content for local development: a verified editor and a verified reader,
// the published articles hello-world and second-post, the draft draft-notes, and two comments by the reader on
// hello-world. The accounts come from the identity module's Provision. The articles and the comments go
// through the application's handler in process, signed in as those accounts, so every rule of the API applies.
//
// Seed skips what already exists, so it can run again: an account whose email is taken, an article whose slug
// is taken, and a comment of the reader on hello-world with the same text. It finishes what an interrupted run
// left: an article that should be published and is still a draft is published, and the comments that are
// missing are added. However often it runs, it converges on the same content. It returns the first error that
// stops it, such as a password the identity module refuses, an account that exists with another password, or a
// database that cannot be reached.
func Seed(ctx context.Context, config app.Config, deps app.Deps, accounts Accounts) error {
	users, err := identity.New(identity.Deps{
		Pool:    deps.Pool,
		Log:     deps.Log,
		Mail:    deps.Mail,
		Service: config.Platform.Service,
		Config:  config.Identity,
	})
	if err != nil {
		return fmt.Errorf("build the identity module: %w", err)
	}
	handler, err := app.NewHandler(config, deps)
	if err != nil {
		return fmt.Errorf("build the application: %w", err)
	}
	s := &seeder{log: deps.Log, users: users, api: client{handler: handler}}

	if err := s.provision(ctx, accounts); err != nil {
		return err
	}

	editor, err := s.api.signIn(ctx, accounts.EditorEmail, accounts.EditorPassword)
	if err != nil {
		return fmt.Errorf("sign in as the editor: %w", err)
	}
	reader, err := s.api.signIn(ctx, accounts.ReaderEmail, accounts.ReaderPassword)
	if err != nil {
		return fmt.Errorf("sign in as the reader: %w", err)
	}

	if err := s.createArticles(ctx, editor); err != nil {
		return err
	}
	return s.createComments(ctx, reader)
}

func (s *seeder) provision(ctx context.Context, accounts Accounts) error {
	people := []struct {
		name  string
		input usecase.ProvisionInput
	}{
		{
			name: "editor",
			input: usecase.ProvisionInput{
				Email:       accounts.EditorEmail,
				Password:    accounts.EditorPassword,
				DisplayName: "Edie Editor",
				Roles:       []string{contract.RoleReader, contract.RoleEditor},
			},
		},
		{
			name: "reader",
			input: usecase.ProvisionInput{
				Email:       accounts.ReaderEmail,
				Password:    accounts.ReaderPassword,
				DisplayName: "Rae Reader",
				Roles:       []string{contract.RoleReader},
			},
		},
	}

	for _, person := range people {
		_, err := s.users.Provision(ctx, person.input)
		if errors.Is(err, domain.ErrEmailTaken) {
			s.log.InfoContext(ctx, "account exists", slog.String("account", person.name))
			continue
		}
		if err != nil {
			return fmt.Errorf("provision the %s: %w", person.name, err)
		}
		s.log.InfoContext(ctx, "account created", slog.String("account", person.name))
	}
	return nil
}

type articleSeed struct {
	slug, title, body string
	publish           bool
}

func (s *seeder) createArticles(ctx context.Context, editor string) error {
	articles := []articleSeed{
		{
			slug:    helloWorld,
			title:   "Hello, world",
			body:    "Welcome to the blog. This is the first published article.",
			publish: true,
		},
		{
			slug:    "second-post",
			title:   "A second post",
			body:    "Another published article, so that the list holds more than one.",
			publish: true,
		},
		{
			slug:  "draft-notes",
			title: "Draft notes",
			body:  "Notes for an article that is not ready; only an editor sees it.",
		},
	}

	for _, article := range articles {
		if err := s.createArticle(ctx, editor, article); err != nil {
			return fmt.Errorf("article %s: %w", article.slug, err)
		}
	}
	return nil
}

func (s *seeder) createArticle(ctx context.Context, editor string, article articleSeed) error {
	var created struct {
		ID string `json:"id"`
	}
	request := map[string]string{"slug": article.slug, "title": article.title, "body": article.body}
	err := s.api.do(ctx, editor, http.MethodPost, "/v1/articles", request, &created, http.StatusCreated)
	if hasProblem(err, "slug_taken") {
		return s.finishArticle(ctx, editor, article)
	}
	if err != nil {
		return err
	}
	if !article.publish {
		s.log.InfoContext(ctx, "draft created", slog.String("slug", article.slug))
		return nil
	}
	return s.publishArticle(ctx, editor, article.slug, created.ID)
}

// finishArticle completes an article that already exists, as an interrupted run leaves it: when the article
// should be published and is still a draft, it publishes it.
func (s *seeder) finishArticle(ctx context.Context, editor string, article articleSeed) error {
	draftID, isDraft, err := s.findArticle(ctx, editor, "/v1/articles/drafts", article.slug)
	if err != nil {
		return err
	}
	if !isDraft || !article.publish {
		s.log.InfoContext(ctx, "article exists", slog.String("slug", article.slug))
		return nil
	}
	return s.publishArticle(ctx, editor, article.slug, draftID)
}

func (s *seeder) publishArticle(ctx context.Context, editor, slug, id string) error {
	err := s.api.do(ctx, editor, http.MethodPost, "/v1/articles/"+id+"/publish", nil, nil, http.StatusOK)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	s.log.InfoContext(ctx, "article published", slog.String("slug", slug))
	return nil
}

func (s *seeder) createComments(ctx context.Context, reader string) error {
	articleID, published, err := s.findArticle(ctx, "", "/v1/articles", helloWorld)
	if err != nil {
		return err
	}
	if !published {
		return fmt.Errorf("article %s is not published", helloWorld)
	}
	var me struct {
		ID string `json:"id"`
	}
	if err := s.api.do(ctx, reader, http.MethodGet, "/v1/users/me", nil, &me, http.StatusOK); err != nil {
		return fmt.Errorf("read the reader's profile: %w", err)
	}
	existing, err := s.commentBodies(ctx, articleID, me.ID)
	if err != nil {
		return err
	}

	bodies := []string{
		"Great start, looking forward to more.",
		"Thanks for writing this.",
	}
	for _, body := range bodies {
		if slices.Contains(existing, body) {
			s.log.InfoContext(ctx, "comment exists", slog.String("slug", helloWorld))
			continue
		}
		request := map[string]string{"article_id": articleID, "body": body}
		if err := s.api.do(ctx, reader, http.MethodPost, "/v1/comments", request, nil, http.StatusCreated); err != nil {
			return fmt.Errorf("comment on article %s: %w", helloWorld, err)
		}
		s.log.InfoContext(ctx, "comment created", slog.String("slug", helloWorld))
	}
	return nil
}

// findArticle returns the ID of the article with the given slug on the list at path, which it reads page by page
// with token as the bearer token unless token is "", and whether the list holds it.
func (s *seeder) findArticle(ctx context.Context, token, path, slug string) (string, bool, error) {
	offset := 0
	for {
		var page struct {
			Items []struct {
				ID   string `json:"id"`
				Slug string `json:"slug"`
			} `json:"items"`
			Total int `json:"total"`
		}
		query := url.Values{"limit": {"100"}, "offset": {strconv.Itoa(offset)}}
		err := s.api.do(ctx, token, http.MethodGet, path+"?"+query.Encode(), nil, &page, http.StatusOK)
		if err != nil {
			return "", false, fmt.Errorf("list %s: %w", path, err)
		}

		for _, item := range page.Items {
			if item.Slug == slug {
				return item.ID, true, nil
			}
		}
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			return "", false, nil
		}
	}
}

// commentBodies returns the text of the comments by authorID on articleID, newest first, up to one page.
func (s *seeder) commentBodies(ctx context.Context, articleID, authorID string) ([]string, error) {
	var page struct {
		Items []struct {
			Body string `json:"body"`
		} `json:"items"`
	}
	query := url.Values{"article_id": {articleID}, "author_id": {authorID}, "limit": {"100"}}
	err := s.api.do(ctx, "", http.MethodGet, "/v1/comments?"+query.Encode(), nil, &page, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("list the reader's comments: %w", err)
	}

	bodies := make([]string, len(page.Items))
	for i, item := range page.Items {
		bodies[i] = item.Body
	}
	return bodies, nil
}
