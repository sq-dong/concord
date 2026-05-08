package blocks

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/epfl-dcsl/schedsim/engine"
)

const (
	bUCKETCOUNT = 100000
	gRANULARITY = 0.01
)

// RequestDrain describes the behaviour of a the element that receives a request
// after processor serving and is in charge of keeping the statistics
type RequestDrain interface {
	TerminateReq(r engine.ReqInterface)
	SetName(name string)
}

// AllKeeper implements the RequestDrain interface and caclulates statistics
// on all the given requests, without sampling
type AllKeeper struct {
	latencies   []float64
	slowdowns   []float64
	name        string
	stolenCount int
}

// TerminateReq is the function called by the processor after finishing
// request processing
func (k *AllKeeper) TerminateReq(req engine.ReqInterface) {
	d := req.GetDelay()
	s := req.GetServiceTime()
	k.latencies = append(k.latencies, d)
	k.slowdowns = append(k.slowdowns, d/s)
	if stealable, ok := req.(*StealableReq); ok {
		if stealable.stolen {
			k.stolenCount++
		}
	}
}

// SetName gives a name to the particular AllKeeper
func (k *AllKeeper) SetName(name string) {
	k.name = name
}

func avg(s []float64) float64 {
	tmp := 0.0
	for _, v := range s {
		tmp += v
	}
	return tmp / float64(len(s))
}

func std(s []float64) float64 {
	tmp := 0.0
	for _, v := range s {
		tmp += v * v
	}
	return math.Sqrt((tmp/float64(len(s)) - avg(s)))
}

func getPercentiles(s []float64) map[float64]float64 {
	res := make(map[float64]float64)
	sort.Float64s(s)
	for _, v := range []float64{0.5, 0.9, 0.95, 0.99} {
		idx := int(float64(len(s)) * v)
		res[v] = s[idx]
	}
	return res
}

// PrintStats prints the collected statistics at the end of the similation.
// This is called by the model
func (k *AllKeeper) PrintStats() {
	fmt.Printf("Stats collector: %v\n", k.name)
	fmt.Printf("Count\tStolen\tAVG (latency)\tSTDDev (latency)\t50th (latency)\t90th (latency)\t95th (latency)\t99th (latency) \tAVG (slowdown)\tSTDDev (slowdown)\t50th (slowdown)\t90th (slowdown)\t95th (slowdown)\t99th (slowdown) \tReqs/time_unit\n")
	fmt.Printf("%v\t%v\t%v\t%v\t", len(k.latencies), k.stolenCount, avg(k.latencies), std(k.latencies))

	vals := []float64{0.5, 0.9, 0.95, 0.99}
	if len(k.latencies) > 0 {
		// FIXME: Leaves a slightly inconsistent print if no data is collected by printing avg for latency and not slowdowns, but is largely irrelevant.
		percentiles := getPercentiles(k.latencies)
		for _, v := range vals {
			fmt.Printf("%v\t", percentiles[v])
		}
		fmt.Printf("%v\t%v\t", avg(k.slowdowns), std(k.slowdowns))
		percentiles = getPercentiles(k.slowdowns)
		for _, v := range vals {
			fmt.Printf("%v\t", percentiles[v])
		}
	}
	fmt.Printf("%v\n", float64(len(k.latencies))/engine.GetTime())
}

// DetailedKeeper maintains two kind of statistics. One is the overall statistics, the rest is per-request.
// Each statistic is maintained in an individual AllKeeper 
type DetailedKeeper struct {
	overall        AllKeeper
	perReqKeepers  []AllKeeper // One AllKeeper per request type
	serviceTimes   []float64   // Service time is used to classify request into types
	name           string
	reqTypes       int
}

// TerminateReq is the function called by the processor after finishing
// request processing
func (k *DetailedKeeper) TerminateReq(req engine.ReqInterface) {
	k.overall.TerminateReq(req)
	if (k.reqTypes > 1) {
		// FIXME: This is a major hack and utilizes the fact that schedsim currently only supports multimodal distributions 
		// by generating service times equal to the peaks. It doesn't work for the exponential distribution, but that's OK,
		// since it has only 1 req type. This should be fixed by embedding the request type in the request itself.
	  s := req.GetServiceTime()
	  p := k.reqTypes
		v := 0.01 // Wasted. In there because of go inexperience
	  for p, v = range k.serviceTimes {
		  if (v == s) {
				break
		  }
    }
	  if (p >= k.reqTypes) {
		  panic("Invalid request received by stats manager")
	  }
	  k.perReqKeepers[p].TerminateReq(req)
	}
}

// SetName gives a name to the particular DetailedKeeper
func (k *DetailedKeeper) SetName(name string) {
	k.name = name
}

// NewDetailedKeeper returns a new *DetailedKeeper
func NewDetailedKeeper(types int, serviceTimes []float64) *DetailedKeeper {
	if (len(serviceTimes) != types) {
		panic("Illegal arguments passed to NewDetailedKeeper")
	}
	k := &DetailedKeeper{}
	k.reqTypes = types
	k.overall.SetName("Overall stats")
	for i := 0; i < types; i++ {
		k.perReqKeepers = append(k.perReqKeepers, AllKeeper{})
		k.serviceTimes = append(k.serviceTimes, serviceTimes[i])
		k.perReqKeepers[i].SetName("Keeper" + strconv.Itoa(i))
	}
	return k
}

// PrintStats prints the collected statistics at the end of the similation.
// This is called by the model
func (k *DetailedKeeper) PrintStats() {
	k.overall.PrintStats()
	if (k.reqTypes > 1) {
		// If only 1 request type, then no difference in overall and per-req type stats
		for i := 0; i < k.reqTypes; i++ {
			k.perReqKeepers[i].PrintStats()
		}
	}
}

// MonitorKeeper keeps statistics about queue lengths
type MonitorKeeper struct {
	delays   []float64
	initLen  []int
	finalLen []int
	name     string
}

// TerminateReq is the function called by the processor after finishing
// request processing
func (k *MonitorKeeper) TerminateReq(req engine.ReqInterface) {
	k.delays = append(k.delays, req.GetDelay())

	if monitorReq, ok := req.(*MonitorReq); ok {
		k.initLen = append(k.initLen, monitorReq.getInitLen())
		k.finalLen = append(k.finalLen, monitorReq.getFinalLen())
	}
}

// PrintStats prints the collected statistics at the end of the similation.
// This is called by the model
func (k *MonitorKeeper) PrintStats() {
	fmt.Println("#Latency\tEntrace Queue\tExit Queue")
	for idx, d := range k.delays {
		fmt.Printf("%v\t%v\t%v\n", d, k.initLen[idx], k.finalLen[idx])
	}
}

// SetName gives a name to the particular AllKeeper
func (k *MonitorKeeper) SetName(name string) {
	k.name = name
}

type histogram struct {
	granularity float64
	buckets     []int
	count       int64
	minBucket   int
	maxBucket   int
	sum         float64
	sumSquare   float64
}

func newHistogram() *histogram {
	return &histogram{
		granularity: gRANULARITY,
		buckets:     make([]int, bUCKETCOUNT),
		minBucket:   bUCKETCOUNT - 1,
		maxBucket:   0,
	}
}

func (hdr *histogram) addSample(s float64) {
	index := int(s / hdr.granularity)
	if index >= bUCKETCOUNT {
		index = bUCKETCOUNT - 1
	}
	if index < 0 || index >= bUCKETCOUNT {
		panic(fmt.Sprintf("Wrong index: %v\n", index))
	}
	hdr.buckets[index]++
	if index > hdr.maxBucket {
		hdr.maxBucket = index
	}
	if index < hdr.minBucket {
		hdr.minBucket = index
	}
	hdr.count++
	hdr.sum += s
	hdr.sumSquare += s * s
}

func (hdr *histogram) avg() float64 {
	return hdr.sum / float64(hdr.count)
}

func (hdr *histogram) stddev() float64 {
	squareAvg := hdr.sumSquare / float64(hdr.count)
	mean := hdr.avg()

	return math.Sqrt(squareAvg - mean*mean)
}

//FIXME: I assume that in every bucket there will be max one percentile
func (hdr *histogram) getPercentiles() map[float64]float64 {
	accum := make([]int, bUCKETCOUNT)
	res := map[float64]float64{}
	percentiles := []float64{0.5, 0.9, 0.95, 0.99}
	percentileI := 0

	accum[hdr.minBucket] = hdr.buckets[hdr.minBucket]

	// what if percentiles in the first bucket
	for float64(accum[hdr.minBucket]) > percentiles[percentileI]*float64(hdr.count) {
		// linear interpolation
		res[percentiles[percentileI]] = hdr.granularity / float64(hdr.buckets[hdr.minBucket]) * (percentiles[percentileI] * float64(hdr.count))
		percentileI++
	}
	if percentileI >= len(percentiles) {
		return res
	}

	for i := hdr.minBucket + 1; i <= hdr.maxBucket; i++ {
		accum[i] = accum[i-1] + hdr.buckets[i]
		for float64(accum[i]) > percentiles[percentileI]*float64(hdr.count) {
			// linear interpolation
			down := hdr.granularity * float64(i-1)

			res[percentiles[percentileI]] = down + hdr.granularity/float64(hdr.buckets[i])*(percentiles[percentileI]*float64(hdr.count)-float64(accum[i-1]))
			percentileI++
			if percentileI >= len(percentiles) {
				return res
			}
		}
	}
	return res
}

func (hdr *histogram) printPercentiles() {
	percentiles := hdr.getPercentiles()
	vals := []float64{0.5, 0.9, 0.95, 0.99}
	for _, v := range vals {
		fmt.Printf("%vth: %v\t", int(v*100.0), percentiles[v])
	}
	fmt.Println()

	fmt.Printf("Req/time_unit:%v\n", float64(hdr.count)/engine.GetTime())
}

// BookKeeper uses buckets to keep the information
type BookKeeper struct {
	hdr  *histogram
	name string
}

// NewBookKeeper returns a new *BookKeeper
func NewBookKeeper() *BookKeeper {
	return &BookKeeper{
		hdr: newHistogram(),
	}
}

// SetName gives a name to the particular AllKeeper
func (b *BookKeeper) SetName(name string) {
	b.name = name
}

// TerminateReq is the function called by the processor after finishing
// request processing
func (b *BookKeeper) TerminateReq(req engine.ReqInterface) {
	d := req.GetDelay()
	b.hdr.addSample(d)
}

// PrintStats prints the collected statistics at the end of the similation.
// This is called by the model
func (b *BookKeeper) PrintStats() {
	fmt.Printf("Stats collector: %v\n", b.name)
	fmt.Printf("Count\tAVG\tSTDDev\t50th\t90th\t95th\t99th Reqs/time_unit\n")
	fmt.Printf("%v\t%v\t%v\t", b.hdr.count, b.hdr.avg(), b.hdr.stddev())

	vals := []float64{0.5, 0.9, 0.95, 0.99}
	percentiles := b.hdr.getPercentiles()
	for _, v := range vals {
		fmt.Printf("%v\t", percentiles[v])
	}
	fmt.Printf("%v\n", float64(b.hdr.count)/engine.GetTime())
}
