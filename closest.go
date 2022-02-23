package flags

func levenshtein(str string, tgt string) int {
	if len(str) == 0 {
		return len(tgt)
	}

	if len(tgt) == 0 {
		return len(str)
	}

	dists := make([][]int, len(str)+1)
	for i := range dists {
		dists[i] = make([]int, len(tgt)+1)
		dists[i][0] = i
	}

	for j := range tgt {
		dists[0][j] = j
	}

	for sidx, sc := range str {
		for tidx, tc := range tgt {
			if sc == tc {
				dists[sidx+1][tidx+1] = dists[sidx][tidx]
			} else {
				dists[sidx+1][tidx+1] = dists[sidx][tidx] + 1
				if dists[sidx+1][tidx] < dists[sidx+1][tidx+1] {
					dists[sidx+1][tidx+1] = dists[sidx+1][tidx] + 1
				}
				if dists[sidx][tidx+1] < dists[sidx+1][tidx+1] {
					dists[sidx+1][tidx+1] = dists[sidx][tidx+1] + 1
				}
			}
		}
	}

	return dists[len(str)][len(tgt)]
}

func closestChoice(cmd string, choices []string) (string, int) {
	if len(choices) == 0 {
		return "", 0
	}

	mincmd := -1
	mindist := -1

	for i, c := range choices {
		l := levenshtein(cmd, c)

		if mincmd < 0 || l < mindist {
			mindist = l
			mincmd = i
		}
	}

	return choices[mincmd], mindist
}
