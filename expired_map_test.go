package expiredmap

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

type cacheItem struct {
	itemName  string
	itemValue int64
}

func TestNewExpiredMap(t *testing.T) {
	em := NewExpiredMap[int64, *cacheItem](10, time.Second)

	// Test Set
	testSet(em, t)

	// Test Get
	testGet(em, t)

	// Test Size
	testSize(em, t)

	// Test capacity
	testCapacity(em, t)

	// Test Foreach
	testForeach(em, t)

	// Test TTL
	testTTL(em, t, time.Second*3)

	// Test Delete
	testDelete(em, t)

	// Test expired
	testExpired(em, t, time.Second*6, time.Second*3)

	// Test New Set
	testNewSet(em, t, time.Second*5)

	// Test concurrency
	cnt := int64(10000 * 100)
	em2 := NewExpiredMap[int64, *cacheItem](cnt, time.Second)
	wg := &sync.WaitGroup{}
	beginTime := time.Now()
	testConcurrency1(em2, t, wg, cnt)
	testConcurrency2(em2, wg, cnt)
	wg.Wait()
	fmt.Println(fmt.Sprintf("%dw read & write speed %d milliseconds", cnt/10000, time.Now().Sub(beginTime).Milliseconds()))

	testClose(em2, t)
}

func testSet(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	for i := int64(1); i <= 10; i++ {
		cacheItemObj := &cacheItem{
			itemName:  fmt.Sprintf("item:%d", i),
			itemValue: i,
		}
		setRes := em.Set(i, cacheItemObj, time.Second*10)
		if !setRes {
			t.Fatalf("Set error")
		}
	}
}

func testGet(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	val, ok := em.Get(1)
	fmt.Println("Get 1, val:", val, "ok:", ok)
	if val == nil || !ok || val.itemValue != 1 {
		t.Fatalf("Get 1 error")
	}
	val2, ok2 := em.Get(2)
	fmt.Println("Get 2, val:", val2, "ok:", ok2)
	if val2 == nil || !ok2 || val2.itemValue != 2 {
		t.Fatalf("Get 2 error")
	}
	val3, ok3 := em.Get(100)
	fmt.Println("Get 100, val:", val3, "ok:", ok3)
	if val3 != nil || ok3 {
		t.Fatalf("Get 100 error")
	}
}

func testSize(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	size := em.Size()
	fmt.Println("Size:", size)
	if size != 10 {
		t.Fatalf("Size error")
	}
}

func testCapacity(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	setRes := em.Set(11, &cacheItem{
		itemName:  fmt.Sprintf("item:%d", 11),
		itemValue: 11,
	}, time.Second*10)
	if setRes == true {
		t.Fatalf("capacity error")
	}
}

func testForeach(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	em.HandleForeach(func(k int64, item *cacheItem) {
		if item == nil {
			t.Fatalf("item nil error")
			return
		}
		if item.itemValue != k {
			t.Fatalf("item value error")
			return
		}
		fmt.Println("itemName:", item.itemName, "itemValue", item.itemValue)
	})
}

func testTTL(em *ExpiredMap[int64, *cacheItem], t *testing.T, duration time.Duration) {
	time.Sleep(duration)
	ttl := em.TTL(1)
	fmt.Println("TTL 1:", ttl.Seconds())
	if int(ttl.Seconds()) != 6 {
		t.Fatalf("TTL 1 error")
	}
}

func testDelete(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	em.Delete(1)
	val, ok := em.Get(1)
	size := em.Size()
	fmt.Println("Delete, Get 1, val:", val, "ok:", ok)
	fmt.Println("Size:", size)
	if val != nil || ok {
		t.Fatalf("Delete, Get 1 error")
	}
	if size != 9 {
		t.Fatalf("Delete, size error")
	}
}

func testExpired(em *ExpiredMap[int64, *cacheItem], t *testing.T, duration1 time.Duration, duration2 time.Duration) {
	time.Sleep(duration1)
	size := em.Size()
	fmt.Println("Size:", size)
	if size != 9 {
		t.Fatalf("Sleep 9s, size error")
	}
	time.Sleep(duration2)
	size = em.Size()
	fmt.Println("Size:", size)
	if size != 0 {
		t.Fatalf("Sleep 10s, size error")
	}
}

func testNewSet(em *ExpiredMap[int64, *cacheItem], t *testing.T, duration time.Duration) {
	em.Set(1, &cacheItem{
		itemName:  fmt.Sprintf("new item:%d", 1),
		itemValue: 1,
	}, time.Second*3)
	size := em.Size()
	fmt.Println("Size:", size)
	if size != 1 {
		t.Fatalf("size error")
	}
	val, ok := em.Get(1)
	if !ok || val == nil || val.itemValue != 1 || val.itemName != fmt.Sprintf("new item:%d", 1) {
		t.Fatalf("Get new 1 error")
	}
	time.Sleep(duration)
}

func testConcurrency1(em *ExpiredMap[int64, *cacheItem], t *testing.T, wg *sync.WaitGroup, cnt int64) {
	for i := int64(0); i < cnt; i++ {
		oneI := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			setRes := em.Set(oneI, &cacheItem{
				itemName:  fmt.Sprintf("new item:%d", oneI),
				itemValue: oneI,
			}, time.Second*60)
			if !setRes {
				t.Errorf("Set error")
				return
			}
		}()
	}
}

func testConcurrency2(em *ExpiredMap[int64, *cacheItem], wg *sync.WaitGroup, cnt int64) {
	for i := int64(0); i < cnt; i++ {
		oneI := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = em.Get(oneI)
		}()
	}
}

func testClose(em *ExpiredMap[int64, *cacheItem], t *testing.T) {
	beginSize := em.Size()
	fmt.Println("beginSize:", beginSize)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Println(fmt.Sprintf("Alloc: %dMb", mem.Alloc/1024/1024))

	em.Close()

	endSize := em.Size()
	fmt.Println("endSize:", endSize)
	if endSize != 0 {
		t.Fatalf("Close error")
	}
	time.Sleep(time.Second * 10)
}
