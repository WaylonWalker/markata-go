package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/services"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(searchCmd())
}

func searchCmd() *cobra.Command {
	var (
		format string
		sortBy string
		order  string
		filter string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across posts",
		Long: `Search post content, titles, descriptions, and tags.

Returns posts matching the query string, sorted by relevance fields.
Useful for agents and scripts to discover content without grep.

Examples:
  markata-go search golang
  markata-go search "error handling" --format json
  markata-go search docker --filter "published == True" --limit 10
  markata-go search cli --sort date --order desc
  markata-go search kubernetes --format path`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			app, err := loadListApp(cmd.Context())
			if err != nil {
				return err
			}

			outputFormat, err := parseListFormat(format)
			if err != nil {
				return err
			}

			searchOpts := services.SearchOptions{
				Limit: limit,
			}

			results, err := app.Posts.Search(cmd.Context(), query, searchOpts)
			if err != nil {
				return err
			}

			// Apply additional filter if provided
			if filter != "" {
				listOpts := services.ListOptions{Filter: filter}
				allFiltered, err := app.Posts.List(cmd.Context(), listOpts)
				if err != nil {
					return err
				}
				results = intersectPosts(results, allFiltered)
			}

			// Apply sorting (default: date desc)
			if sortBy == "" {
				sortBy = "date"
			}
			sortOrder, err := parseSortOrder(order)
			if err != nil {
				return err
			}
			if !isValidPostSort(sortBy) {
				return fmt.Errorf("invalid sort field %q", sortBy)
			}
			sortSearchResults(results, sortBy, sortOrder)

			if outputFormat == listFormatTable {
				return renderSearchTable(results, query)
			}
			return renderPosts(outputFormat, results)
		},
	}

	cmd.Flags().StringVar(&format, "format", listFormatTable, "output format: table, json, csv, path")
	cmd.Flags().StringVar(&sortBy, "sort", "date", "sort field: date, title, words, path, reading_time, tags")
	cmd.Flags().StringVar(&order, "order", "desc", "sort order: asc or desc")
	cmd.Flags().StringVar(&filter, "filter", "", "additional filter expression")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of results (0 = no limit)")

	return cmd
}

// sortSearchResults sorts posts by the given field and order.
func sortSearchResults(posts []*models.Post, field string, order services.SortOrder) {
	sort.SliceStable(posts, func(i, j int) bool {
		var cmp int
		switch strings.ToLower(field) {
		case "date":
			cmp = compareDatePtrs(posts[i].Date, posts[j].Date)
		case "title":
			cmp = strings.Compare(
				strings.ToLower(derefStr(posts[i].Title)),
				strings.ToLower(derefStr(posts[j].Title)),
			)
		case "path":
			cmp = strings.Compare(posts[i].Path, posts[j].Path)
		case "words":
			cmp = compareInts(postWordCount(posts[i]), postWordCount(posts[j]))
		case "reading_time":
			cmp = compareInts(postReadingTime(posts[i]), postReadingTime(posts[j]))
		case "tags":
			cmp = strings.Compare(
				strings.ToLower(strings.Join(posts[i].Tags, ", ")),
				strings.ToLower(strings.Join(posts[j].Tags, ", ")),
			)
		}
		if cmp == 0 {
			cmp = strings.Compare(posts[i].Path, posts[j].Path)
		}
		if order == services.SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareDatePtrs(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if a.Before(*b) {
		return -1
	}
	if a.After(*b) {
		return 1
	}
	return 0
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// intersectPosts returns posts present in both slices, preserving order of a.
func intersectPosts(a, b []*models.Post) []*models.Post {
	set := make(map[string]struct{}, len(b))
	for _, p := range b {
		set[p.Path] = struct{}{}
	}
	result := make([]*models.Post, 0, len(a))
	for _, p := range a {
		if _, ok := set[p.Path]; ok {
			result = append(result, p)
		}
	}
	return result
}

// renderSearchTable renders search results with a match count header.
func renderSearchTable(posts []*models.Post, query string) error {
	outlnf("%d results for %q\n", len(posts), query)
	if len(posts) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(outWriter(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TITLE\tDATE\tWORDS\tREAD\tTAGS\tPATH")
	fmt.Fprintln(w, "-----\t----\t-----\t----\t----\t----")
	for _, post := range posts {
		row := postToRow(post)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Title,
			row.Date,
			formatWordCount(row.Words),
			formatReadingTime(row.ReadingTime),
			strings.Join(row.Tags, ", "),
			row.Path,
		)
	}
	return w.Flush()
}
