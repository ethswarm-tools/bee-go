// swarm-blog is a markdown-driven blog with a single stable URL.
//
// Posts live as posts/<slug>.md files locally. On `publish`, each post
// is wrapped in a tiny HTML shell, an index.html listing is generated,
// and the whole site is uploaded as a Mantaray collection whose root
// is published through a feed manifest. The feed manifest URL stays
// the same forever; readers always see the latest version.
//
// Usage:
//
//	swarm-blog init  <title>
//	swarm-blog new   <slug> <title>
//	swarm-blog list
//	swarm-blog publish
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const (
	blogFile = "_blog.json"
	postsDir = "posts"
)

type blogState struct {
	Title           string `json:"title"`
	TopicHex        string `json:"topic_hex"`
	OwnerHex        string `json:"owner_hex"`
	FeedManifestRef string `json:"feed_manifest_ref"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-blog <init|new|list|publish> ...")
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	switch args[0] {
	case "init":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-blog init <title>")
		}
		return cmdInit(client, url, args[1])
	case "new":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-blog new <slug> <title>")
		}
		return cmdNew(args[1], args[2])
	case "list":
		return cmdList()
	case "publish":
		return cmdPublish(client, url)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdInit(client *bee.Client, url, title string) error {
	if _, err := os.Stat(blogFile); err == nil {
		return fmt.Errorf("%s already exists", blogFile)
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	signer, err := envSigner()
	if err != nil {
		return err
	}
	owner := signer.PublicKey().Address()
	topic := swarm.TopicFromString("swarm-blog:" + title)

	feedManifest, err := client.File.CreateFeedManifest(context.Background(), batchID, owner, topic)
	if err != nil {
		return fmt.Errorf("create_feed_manifest: %w", err)
	}

	if err := os.MkdirAll(postsDir, 0755); err != nil {
		return fmt.Errorf("mkdir posts: %w", err)
	}
	st := blogState{
		Title:           title,
		TopicHex:        topic.Hex(),
		OwnerHex:        owner.Hex(),
		FeedManifestRef: feedManifest.Hex(),
	}
	if err := saveState(&st); err != nil {
		return err
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Initialised blog %q\n", title)
	fmt.Printf("  feed manifest: %s\n", feedManifest.Hex())
	fmt.Printf("  stable URL:    %s/bzz/%s/\n", trimmed, feedManifest.Hex())
	fmt.Println("\nNext: `swarm-blog new <slug> <title>` then `swarm-blog publish`.")
	return nil
}

func cmdNew(slug, title string) error {
	if _, err := loadState(); err != nil {
		return err
	}
	path := filepath.Join(postsDir, slug+".md")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	template := fmt.Sprintf(
		"# %s\n\nWrite your post here. Markdown is preserved\nin a <pre> block on publish; this is a starter template.\n",
		title)
	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	fmt.Printf("Created %s\n", path)
	return nil
}

func cmdList() error {
	posts, err := listPosts()
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		fmt.Println("(no posts yet — `swarm-blog new <slug> <title>`)")
		return nil
	}
	fmt.Println("posts/")
	for _, p := range posts {
		fmt.Printf("  %-24s %s\n", p.slug, p.title)
	}
	return nil
}

func cmdPublish(client *bee.Client, url string) error {
	st, err := loadState()
	if err != nil {
		return err
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	signer, err := envSigner()
	if err != nil {
		return err
	}
	topic, err := swarm.TopicFromHex(st.TopicHex)
	if err != nil {
		return fmt.Errorf("parse topic: %w", err)
	}

	posts, err := listPosts()
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return fmt.Errorf("no posts in posts/ — nothing to publish")
	}

	var entries []file.CollectionEntry
	var links strings.Builder
	for _, p := range posts {
		mdPath := filepath.Join(postsDir, p.slug+".md")
		body, err := os.ReadFile(mdPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", mdPath, err)
		}
		entries = append(entries, file.CollectionEntry{
			Path: "posts/" + p.slug + ".html",
			Data: []byte(postHTML(p.title, string(body))),
		})
		fmt.Fprintf(&links, "<li><a href=\"posts/%s.html\">%s</a></li>\n", p.slug, p.title)
	}
	entries = append(entries, file.CollectionEntry{
		Path: "index.html",
		Data: []byte(indexHTML(st.Title, links.String())),
	})

	fmt.Printf("Uploading %d entries...\n", len(entries))
	opts := &api.CollectionUploadOptions{IndexDocument: "index.html"}
	result, err := client.File.UploadCollectionEntries(context.Background(), batchID, entries, opts)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	fmt.Printf("  site ref: %s\n", result.Reference.Hex())

	fmt.Println("Updating feed pointer...")
	if _, err := client.File.UpdateFeedWithReference(context.Background(),
		batchID, signer, topic, result.Reference, nil); err != nil {
		return fmt.Errorf("update_feed: %w", err)
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("\nPublished %d posts.\n", len(posts))
	fmt.Printf("  stable URL: %s/bzz/%s/\n", trimmed, st.FeedManifestRef)
	return nil
}

type postInfo struct{ slug, title string }

func listPosts() ([]postInfo, error) {
	if _, err := loadState(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(postsDir); os.IsNotExist(err) {
		return nil, nil
	}
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return nil, fmt.Errorf("read posts: %w", err)
	}
	var out []postInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		body, _ := os.ReadFile(filepath.Join(postsDir, e.Name()))
		title := slug
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				break
			}
		}
		out = append(out, postInfo{slug: slug, title: title})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].slug < out[j].slug })
	return out, nil
}

func indexHTML(title, links string) string {
	return fmt.Sprintf(
		"<!doctype html>\n<html><head><meta charset=\"utf-8\">\n<title>%s</title></head><body>\n<h1>%s</h1>\n<ul>\n%s</ul>\n<hr><p><small>powered by swarm-blog</small></p>\n</body></html>\n",
		title, title, links)
}

func postHTML(title, bodyMD string) string {
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(bodyMD)
	return fmt.Sprintf(
		"<!doctype html>\n<html><head><meta charset=\"utf-8\">\n<title>%s</title></head><body>\n<p><a href=\"../index.html\">&larr; back</a></p>\n<pre>%s</pre>\n</body></html>\n",
		title, escaped)
}

func envBatch() (swarm.BatchID, error) {
	h := os.Getenv("BEE_BATCH_ID")
	if h == "" {
		return swarm.BatchID{}, fmt.Errorf("BEE_BATCH_ID is required")
	}
	return swarm.BatchIDFromHex(h)
}

func envSigner() (swarm.PrivateKey, error) {
	h := os.Getenv("BEE_SIGNER_HEX")
	if h == "" {
		return swarm.PrivateKey{}, fmt.Errorf("BEE_SIGNER_HEX is required")
	}
	return swarm.PrivateKeyFromHex(h)
}

func saveState(s *blogState) error {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(blogFile, bytes, 0644)
}

func loadState() (*blogState, error) {
	bytes, err := os.ReadFile(blogFile)
	if err != nil {
		return nil, fmt.Errorf("%s not found — run `init` first", blogFile)
	}
	var s blogState
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
