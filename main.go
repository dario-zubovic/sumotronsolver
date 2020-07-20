package main

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	width  = 6
	height = 6

	maximumPosX = 100
	maximumPosY = 100

	maxWorkers = 128
)

var (
	workers int32
)

type position struct {
	x int
	y int
}

type grid map[position]int

type stack []grid

func (s *stack) push(element grid) {
	*s = append(*s, element)
}

func (s *stack) pop() grid {
	element := (*s)[len(*s)-1]
	*s = (*s)[:(len(*s) - 1)]
	return element
}

func sumAtPosition(grid grid, pos position) int {
	sum := 0

	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			if x == 0 && y == 0 {
				continue
			}

			sum += grid[position{
				x: pos.x + x,
				y: pos.y + y,
			}]
		}
	}

	if sum == 0 {
		sum = 1
	}
	return sum
}

func gridDeepCopy(original grid) grid {
	clone := make(grid)

	for k, v := range original {
		clone[k] = v
	}

	return clone
}

func exploreFrom(startingGrid grid, wg *sync.WaitGroup, resultCh chan grid) {
	wg.Add(1)
	defer wg.Done()

	stack := make(stack, 0)
	stack.push(startingGrid)

	maximum := 0

	for len(stack) > 0 {
		grid := stack.pop()

		highestSum := 0
		var highestSumPositions []position
		secondHighestSum := 0
		var secondHighestSumPositions []position

		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				pos := position{
					x: x,
					y: y,
				}

				if grid[pos] > 0 {
					continue
				}

				sum := sumAtPosition(grid, pos)

				if sum > highestSum {
					secondHighestSum = highestSum
					secondHighestSumPositions = highestSumPositions

					highestSum = sum
					highestSumPositions = make([]position, 0)
					highestSumPositions = append(highestSumPositions, pos)
				} else if sum == highestSum {
					highestSumPositions = append(highestSumPositions, pos)
				} else if sum > secondHighestSum {
					secondHighestSum = sum
					secondHighestSumPositions = make([]position, 0)
					secondHighestSumPositions = append(secondHighestSumPositions, pos)
				} else if sum == secondHighestSum {
					secondHighestSumPositions = append(secondHighestSumPositions, pos)
				}
			}
		}

		maxVal := grid[position{
			x: maximumPosX,
			y: maximumPosY,
		}]

		if highestSum > 0 {
			for _, pos := range highestSumPositions {
				newGrid := gridDeepCopy(grid)
				newGrid[pos] = highestSum
				if highestSum > maxVal {
					newGrid[position{
						x: maximumPosX,
						y: maximumPosY,
					}] = highestSum
				}
				if spawnNewWorkerTest() {
					go exploreFrom(newGrid, wg, resultCh)
				} else {
					stack.push(newGrid)
				}
			}

			for _, pos := range secondHighestSumPositions {
				if secondHighestSum > highestSum*87/100 ||
					(highestSum < 10 && secondHighestSum > 1) ||
					((pos.x == 0 || pos.x == width-1) && (pos.y == 0 || pos.y == height-1) && secondHighestSum >= maxVal) { // behave non-greedy at grid edges
					newGrid := gridDeepCopy(grid)
					newGrid[pos] = secondHighestSum
					if secondHighestSum > maxVal {
						newGrid[position{
							x: maximumPosX,
							y: maximumPosY,
						}] = secondHighestSum
					}
					if spawnNewWorkerTest() {
						go exploreFrom(newGrid, wg, resultCh)
					} else {
						stack.push(newGrid)
					}
				}
			}
		} else {
			if maxVal > maximum {
				maximum = maxVal
				resultCh <- grid
			}
		}
	}

	atomic.AddInt32(&workers, -1)
}

func spawnNewWorkerTest() bool {
	for i := int32(0); i < maxWorkers; i++ {
		if atomic.CompareAndSwapInt32(&workers, i, i+1) {
			return true
		}
	}

	return false
}

func main() {
	startingGrid := make(grid)
	startingGrid[position{
		x: 0,
		y: 0,
	}] = 1
	startingGrid[position{
		x: 2,
		y: 0,
	}] = 1

	resultsCh := make(chan grid, maxWorkers)
	closeCh := make(chan bool)

	go func() {
		maximum := 0
		var maximumGrid grid

		run := true
		for run {
			select {
			case grid := <-resultsCh:
				value := grid[position{
					x: maximumPosX,
					y: maximumPosY,
				}]
				if value > maximum {
					maximum = value
					maximumGrid = grid
					fmt.Println(maximum)
				}

			case <-closeCh:
				run = false
			}
		}

		for y := 0; y < height; y++ {
			line := ""
			for x := 0; x < width; x++ {
				line += strconv.Itoa(maximumGrid[position{
					x: x,
					y: y,
				}]) + " "
			}
			fmt.Println(line)
		}
	}()

	workers = 1
	wg := &sync.WaitGroup{}

	go exploreFrom(startingGrid, wg, resultsCh)
	time.Sleep(10 * time.Second)

	wg.Wait()

	closeCh <- true
	time.Sleep(10 * time.Second)
}
