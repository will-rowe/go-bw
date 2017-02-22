package main


import (
  "fmt"
  "sort"
  "strings"
  "math"
)


var reference string = "acgacaacgacgtttcgcgctgcgatcgactgcaacgacaacgacg"
var query string = "acga"


// declare a struct for collecting each suffix and the offset
type Suffix struct {
  suffix string
  offset int
}

// define our collection type (so that we can sort a slice of Suffixs using sort.Interface)
type Suffixes []Suffix

// implement the interface by giving the Suffixes type all the methods it needs to satisfy the sort.Interface interface
func (slice Suffixes) Len() int {
    return len(slice)
}
func (slice Suffixes) Less(i, j int) bool {
    return slice[i].suffix < slice[j].suffix;
}
func (slice Suffixes) Swap(i, j int) {
    slice[i], slice[j] = slice[j], slice[i]
}


func main() {
  fmt.Printf("\nquery sequence =\t%s\n", query)
  fmt.Printf("\nreference sequence =\t%s\n\n", reference)
  fmt.Println("----------------------------------------------------------------------------------")
  bw_align(reference)
  fmt.Println("----------------------------------------------------------------------------------")
}


//
// main bw_align function
//
func bw_align(reference string) {

  // 1. append termination symbol to the reference sequence
  if strings.HasSuffix(reference, "$") == false {
    reference = reference + "$"
  }


  // function to generate a suffix array of our reference and return the offsets
  get_SA := func() []int {
    var suffix_array Suffixes
    for i, j := 0, len(reference)-1; i <= j; i++ {
      suffix_array = append(suffix_array, Suffix{suffix: reference[i:j], offset: i})
    }
    // sort our slice of Suffixes lexicographically using our inteface
    sort.Sort(suffix_array)
    // save just the offsets (by row number) and return these
    offset_array := []int{}
    for _, suffix := range suffix_array {
      offset_array = append(offset_array, suffix.offset)
    }
    return offset_array
  }
  // 2. call func to generate suffix array
  offset_array := get_SA()


  // function to take only the suffix array offsets for every nth suffix (downsampling)
  downsampleSuffixArray := func() map[int]int {
    var n float64 = 4.0
    ds_offset_array := make(map[int]int)
    for i := range offset_array {
      if math.Mod(float64(i), n) == 0 {
        ds_offset_array[i] = offset_array[i]
      }
    }
    return ds_offset_array
  }
  // 3. call func to downsample suffix array
  ds_offset_array := downsampleSuffixArray()


  // 4. build the bwt of our reference by taking the character to the left of each suffix start
  var bwt []string
  for _, offset := range offset_array {
    if offset == 0 {
      bwt = append(bwt, "$")
    } else {
      bwt = append(bwt, reference[offset-1:offset])
    }
  }


  // function to create rank check points whilst scanning along the bwt
  createRankCPs := func() map[string][]int {
    var n int = 4
    checkpoints := make(map[string][]int)
    tally := make(map[string]int)
    for _, character := range bwt {
      // add a key in both the checkpoints and tally maps for each distinct character in reference (ACTG)
      if _, ok := tally[character]; ok == false {
        tally[character] = 0
        checkpoints[character] = []int{}
      }
    }
    // cycle through the bwt again, populating the two maps
    for i, character := range bwt {
      // add character to the character tally
      tally[character] += 1
      // create checkpoint for every n bases of bwt
      if math.Mod(float64(i), float64(n)) == 0 {
        for key := range tally {
          checkpoints[key] = append(checkpoints[key], tally[key])
        }
      }
    }
    return checkpoints
  }
  // 5. call func to create checkpoints for ranks
  checkpoints := createRankCPs()


  // 6. calculate the number of occurences for each character in the bwt
  totals := make(map[string]int)
  for _, character := range bwt {
    totals[character] += 1
  }

  // 7. calculate concise representation of the first column of the bw matrix (i.e. first occurence of each char)
  // to sort the totals map, create a sorted array of keys and then iterate over the map using the sorted keys
  keys_from_totals := make([]string, 0)
  for key := range totals {
    keys_from_totals = append(keys_from_totals, key)
  }
  sort.Strings(keys_from_totals)
  first_col_chars := make(map[string]int, len(totals))
  var total_char_count int = 0
  // lowest char lexically will start on 0, then the next will start after the total number of lowest char
  for _, key := range keys_from_totals {
    first_col_chars[key] = total_char_count
    total_char_count += totals[key]
  }


  // function to count the number of occurences of characters lexically smaller than a query char up to its location in btw
  count := func(query_char string) (count int) {
    // in case query not in text
    if _, ok := first_col_chars[query_char]; ok == false {
      for _, key := range keys_from_totals {
        if query_char < key {
          count = first_col_chars[key]
        }
        count = first_col_chars[key]
      }
    } else {
      count = first_col_chars[query_char]
    }
    return
  }


  // function to return the number of characters there are in the bw matrix, including query row
  rank := func(character string, row int) int {
    var n int = 4
    var num_of_chars_from_cp int = 0
    if _, ok := checkpoints[character]; ok == false || row < 0 {
      return 0
    }
    // walk to the left along the bwt (upwards in the bw matrix) when calculating rank
    // check that our query row is not in the checkpoint map (i.e. not divisible exactly by n)
    row_iterator := row
    for j := 0; row_iterator>j; row_iterator -= 1 {
      if math.Mod(float64(row_iterator), float64(n)) != 0 {
        if bwt[row_iterator] == character {
          num_of_chars_from_cp += 1
        }
      } else {
        break
      }
    }
    // return number of query chars in the bwt up to and including search row
    return (checkpoints[character][row_iterator / n] + num_of_chars_from_cp)
  }


  // 8. find occurences of a subsequence (exact matching with FM index)
  // get a range of the bw matrix rows that have the query as a prefix
  l, r := 0, len(bwt)-1
  // l and r delimit the range of row beginning with progressively longer suffixes of the query
  // move right to left through the query sequence
  for i := len(query)-1; i >= 0; i = i - 1 {
    // LF function to match char and rank (LF = num chars lexically smaller than qc in bwt + number of qc chars before position idx in bwt)
    l = rank(string(query[i]), l-1) + count(string(query[i]))
    r = rank(string(query[i]), r) + count(string(query[i])) - 1
    if r < 1 {
      break
    }
  }
  r = r+1


  // function to resolve the location of the query in the reference
  resolve := func(row int) int {

    stepLeft := func(row int) int {
      c := bwt[row]
      return rank(c, row-1) + count(c)
    }

    nsteps := 0
    for row > 0 {
      if _, ok := ds_offset_array[row]; ok == false {
        row = stepLeft(row)
        nsteps += 1
      } else {
        break
      }
    }
    if row == 0 {
      return row
    }
    return ds_offset_array[row] + nsteps
  }


  // 9. resolve the location of the query in the reference
  for i, j := l, r; i<j; i += 1 {
    location_start := resolve(i)
    fmt.Printf("\nreference sequence =\t%s", reference)
    fmt.Printf("\nexact query align =\t")
    for i, j := 0, len(reference)-1; i<j; i = i+1 {
      if i == location_start {
        fmt.Println(query)
        break
      } else {
        fmt.Printf(" ")
      }
    }
  }





}
