/*
 * Copyright (c) 2017, Lee Keitel
 * This file is released under the BSD 3-Clause license.
 *
 * This file demonstrates using a loop to generate the Fibonacci sequence.
 */

const main = func() {
    let count = 1
    let num1 = 0
    let num2 = 1

    for i = 0; num1 >= 0; i += 1 {
        println(count, ": ", num1, " ")

        let sumOfPrevTwo = num1 + num2
        num1 = num2
        num2 = sumOfPrevTwo
        count += 1
    }
}

main()
