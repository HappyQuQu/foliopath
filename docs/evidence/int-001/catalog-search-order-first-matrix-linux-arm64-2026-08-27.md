# Catalog search order-first correctness matrix

Status: **candidate broadened; production fix still not selected**.

The benchmark-only order-first candidate was compared with the current SQLite
repository on native Linux/arm64, 4 CPU/4 GiB, no network and a read-only root
filesystem. The fixture contained 10,000 directories and 100,000 assets. No
production query, schema, API or 250 ms budget changed.

The matrix covers library/global/root-recursive/direct-directory/subtree scopes,
name/modified/size ordering in both applicable directions, non-empty image/video
and animated kind filters, a combined image-plus-video filter, an unanchored
two-character term, a modification window, and keyset page two whenever the
first result contained 101 rows. All 26 first-page and all 22 second-page
candidate ID sequences exactly matched the production repository.

Across the timing/plan runs, candidate one-shot page latency was at most 202.268 ms for page one and 155.230
ms for page two. The slow candidate cases were recursive-directory queries;
all candidate observations stayed below the unchanged isolated 250 ms page
budget. The established broad library query still had complete first-page
thumbnail/storyboard/favorite hydration measured separately; the matrix itself
compared ordered IDs, not every hydrated field.

The current worst kind filter (`library/kind-video`), recursive name ordering
and broad modification-window candidate were then sampled 20 times per page.
Their worst P95 was 174.591 ms on page one and 135.115 ms on page two, also
below 250 ms. This is representative repeated evidence, not a P95 claim for
all 48 matrix pages.

The matrix also exposed a serious existing-path counterexample. Across two
runs, a broad library/name query with a modification window took the production
repository 9.443–18.139 s for page one and 9.545–18.233 s for page two, while
the matching candidate took 20.425–36.262 ms and 33.806–36.197 ms.

The filtered plans now prove the execution shape: `assets_modified` drives the
broad time-range scan, FTS membership is evaluated against that stream, and a
temporary B-tree still sorts the results into folder/name order. Page two has
the same plan. The candidate instead uses `assets_browse_folder_name_v2` and an
FTS `EXISTS` membership check, with no temporary sort. This establishes the
counterexample's root execution shape, but it does not yet define a safe
selectivity threshold for real date windows.

The candidate is therefore stronger but not ready for production. The fixture
had only one library; the matrix ran after storyboard seeding with about 80,000
images, 10,000 videos and 10,000 animated assets. Image, video, animated and the
combined image-plus-video kind paths are covered, but other combined kind sets
are not. The date window was broad; only three representative cases have repeated P95 samples,
full hydration was not repeated across every combination, and native amd64 is
missing. Real-world selectivity, cross-library mixed-media/date/sparse distributions and a
count/selectivity strategy must be tested before choosing one
planner path.

The exact matrices, binary digest and limitations are in the adjacent JSON.
