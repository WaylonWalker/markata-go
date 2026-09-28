// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

const pinsFeedSlug = "pins"

type SubscriptionFeedsPlugin struct { implicitRoot bool }
func NewSubscriptionFeedsPlugin() *SubscriptionFeedsPlugin { return &SubscriptionFeedsPlugin{} }
func (p *SubscriptionFeedsPlugin) Name() string { return "subscription_feeds" }
func (p *SubscriptionFeedsPlugin) Priority(stage lifecycle.Stage) int { if stage == lifecycle.StageCollect { return lifecycle.PriorityEarly }; return lifecycle.PriorityDefault }

// Collect injects built-in subscription feeds into the feed configs.
//nolint:gocyclo // Collection reconciles cached/configured root, archive, and pins ownership.
func (p *SubscriptionFeedsPlugin) Collect(m *lifecycle.Manager) error {
	config := m.Config(); syndication := getSyndicationConfig(config)
	if config.Extra != nil { if disabled, ok := config.Extra["subscription_feeds_disabled"].(bool); ok && disabled { return nil } }
	feedConfigs := getFeedConfigs(config)
	if cached, ok := m.Cache().Get("feed_configs"); ok { if fcs, ok := cached.([]models.FeedConfig); ok { known := make(map[string]bool, len(feedConfigs)); for i := range feedConfigs { known[feedConfigs[i].Slug] = true }; for i := range fcs { if !known[fcs[i].Slug] { known[fcs[i].Slug] = true; feedConfigs = append(feedConfigs, fcs[i]) } } } }
	rootFeedIndex := -1; hasArchiveFeed := false; hasPinsFeed := false
	for i := range feedConfigs { if feedConfigs[i].Slug == "" { rootFeedIndex = i }; if feedConfigs[i].Slug == defaultArchivePrefix { hasArchiveFeed = true }; if feedConfigs[i].Slug == pinsFeedSlug { hasPinsFeed = true } }
	authoredHomepage := hasAuthoredHomepage(m.Posts()); configuredRoot := hasConfiguredRootFeed(config)
	if rootFeedIndex >= 0 && p.implicitRoot && !configuredRoot { feedConfigs[rootFeedIndex].Formats.HTML = !authoredHomepage }; if configuredRoot { p.implicitRoot = false }
	if rootFeedIndex < 0 { feedConfigs = append(feedConfigs, models.FeedConfig{Slug:"", Title:getSubscriptionFeedTitle(config,"root"), Description:getSubscriptionFeedDescription(config,"root"), Filter:"published == true", Sort:"date", Reverse:true, Formats:models.FeedFormats{HTML:!authoredHomepage,RSS:true,Atom:true}}); p.implicitRoot = true }
	if !hasArchiveFeed && !syndication.SiteArchiveDisabled { feedConfigs = append(feedConfigs, models.FeedConfig{Slug:defaultArchivePrefix, Title:getSubscriptionFeedTitle(config,defaultArchivePrefix), Description:getSubscriptionFeedDescription(config,defaultArchivePrefix), Filter:"published == true", Sort:"date", Reverse:true, Formats:models.FeedFormats{RSS:true,Atom:true}}) }
	if !hasPinsFeed && !postOwnsFeedSlug(m.Posts(), pinsFeedSlug) { feedConfigs = append(feedConfigs, models.FeedConfig{Slug:pinsFeedSlug,Title:"Pins",Description:"A field notebook of saved links",Filter:"published == true and link",Sort:"date",Reverse:true,Templates:models.FeedTemplates{HTML:"pins.html"},Formats:models.FeedFormats{HTML:true}}) }
	config.Extra["feeds"] = feedConfigs; m.Cache().Set("feed_configs", feedConfigs); return nil
}

func hasAuthoredHomepage(posts []*models.Post) bool { for _, post := range posts { if post == nil || post.Skip || post.Draft { continue }; if post.Slug == "" { return true } }; return false }
func hasConfiguredRootFeed(config *lifecycle.Config) bool { modelsConfig, ok := getModelsConfig(config); if !ok || modelsConfig == nil { return false }; for i := range modelsConfig.Feeds { if modelsConfig.Feeds[i].Slug == "" { return true } }; return false }
func postOwnsFeedSlug(posts []*models.Post, slug string) bool { slug = strings.Trim(strings.TrimSpace(slug), "/"); if slug == "" { return false }; for _, post := range posts { if post == nil || post.Skip || post.Draft { continue }; if strings.Trim(strings.TrimSpace(post.Slug), "/") == slug { return true } }; return false }
func getSubscriptionFeedTitle(config *lifecycle.Config, feedType string) string { siteTitle := "Site"; if config.Extra != nil { if title, ok := config.Extra["title"].(string); ok && title != "" { siteTitle = title } }; switch feedType { case "root": return siteTitle+" Feed"; case defaultArchivePrefix: return siteTitle+" Archive Feed"; default: return siteTitle+" Feed" } }
func getSubscriptionFeedDescription(config *lifecycle.Config, feedType string) string { siteDescription := ""; if config.Extra != nil { if desc, ok := config.Extra["description"].(string); ok { siteDescription=desc } }; if siteDescription!="" { return siteDescription }; switch feedType { case "root": return "All published posts"; case defaultArchivePrefix: return "Archive of all published posts"; default: return "Posts feed" } }

type DiscoveryFeed struct { Slug string; Title string; RSSURL string; AtomURL string; JSONURL string; HasRSS bool; HasAtom bool; HasJSON bool }
func GetDiscoveryFeed(_ *models.Post, sidebarFeed *models.FeedConfig, allFeeds []models.FeedConfig) *DiscoveryFeed { if sidebarFeed != nil { return feedConfigToDiscoveryFeed(sidebarFeed) }; for i:=range allFeeds { if allFeeds[i].Slug=="" { return feedConfigToDiscoveryFeed(&allFeeds[i]) } }; return &DiscoveryFeed{Slug:"",Title:"Site Feed",RSSURL:"/rss.xml",AtomURL:"/atom.xml",HasRSS:true,HasAtom:true} }
func feedConfigToDiscoveryFeed(fc *models.FeedConfig) *DiscoveryFeed { df:=&DiscoveryFeed{Slug:fc.Slug,Title:fc.Title,HasRSS:fc.Formats.RSS,HasAtom:fc.Formats.Atom,HasJSON:fc.Formats.JSON}; baseURL:=""; if fc.Slug!="" { baseURL="/"+fc.Slug }; if df.HasRSS { df.RSSURL=baseURL+"/rss.xml" }; if df.HasAtom { df.AtomURL=baseURL+"/atom.xml" }; if df.HasJSON { df.JSONURL=baseURL+"/feed.json" }; return df }
func DiscoveryFeedToMap(df *DiscoveryFeed) map[string]interface{} { if df==nil { return nil }; return map[string]interface{}{"slug":df.Slug,"title":df.Title,"rss_url":df.RSSURL,"atom_url":df.AtomURL,"json_url":df.JSONURL,"has_rss":df.HasRSS,"has_atom":df.HasAtom,"has_json":df.HasJSON} }
func GetFeedBySlug(slug string, feedConfigs []models.FeedConfig) *models.FeedConfig { for i:=range feedConfigs { if feedConfigs[i].Slug==slug { return &feedConfigs[i] } }; return nil }
func FindPostSidebarFeed(post *models.Post, config *lifecycle.Config, feedConfigs []models.FeedConfig) *models.FeedConfig { seriesCfg:=parseSeriesConfig(config); components,ok:=config.Extra["components"].(models.ComponentsConfig); if !ok || components.FeedSidebar.Enabled==nil || !*components.FeedSidebar.Enabled { return nil }; feedSlugs:=components.FeedSidebar.Feeds; if len(feedSlugs)==0 { return nil }; for _,feedSlug:=range feedSlugs { if strings.HasPrefix(feedSlug,"tags/") { tagName:=strings.TrimPrefix(feedSlug,"tags/"); for _,postTag:=range post.Tags { if postTag==tagName { return GetFeedBySlug(feedSlug,feedConfigs) } } }; if post.PrevNextFeed==feedSlug { return GetFeedBySlug(feedSlug,feedConfigs) }; if feed,ok:=post.Extra["feed"].(string); ok && feed==feedSlug { return GetFeedBySlug(feedSlug,feedConfigs) }; if series,ok:=post.Extra["series"].(string); ok { seriesSlug:=getStringFromExtra(post.Extra,"series_slug"); if seriesSlug=="" { seriesSlug=buildSeriesFeedSlug(seriesCfg.SlugPrefix,models.Slugify(series)) }; if feedSlug==seriesSlug { return GetFeedBySlug(feedSlug,feedConfigs) } } }; return nil }
var (_ lifecycle.Plugin=(*SubscriptionFeedsPlugin)(nil); _ lifecycle.CollectPlugin=(*SubscriptionFeedsPlugin)(nil); _ lifecycle.PriorityPlugin=(*SubscriptionFeedsPlugin)(nil))
