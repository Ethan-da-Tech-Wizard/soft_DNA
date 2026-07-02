import Foundation

struct Point {
    var x: Double
    var y: Double
}

func distance(from a: Point, to b: Point) -> Double {
    let dx = a.x - b.x
    let dy = a.y - b.y
    return (dx * dx + dy * dy).squareRoot()
}

func main() {
    let p1 = Point(x: 0, y: 0)
    let p2 = Point(x: 3, y: 4)
    print(distance(from: p1, to: p2))
}

main()
