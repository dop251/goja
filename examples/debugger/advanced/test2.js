// Fibonacci example
function fibonacci(n) {
    if (n <= 1) {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

console.log("Calculating Fibonacci...");
var n = 6;
var result = fibonacci(n);
console.log("Fibonacci of", n, "is:", result);

// Array operations
var numbers = [1, 2, 3, 4, 5];
var doubled = [];

for (var i = 0; i < numbers.length; i++) {
    doubled.push(numbers[i] * 2);
}

console.log("Original:", numbers);
console.log("Doubled:", doubled);