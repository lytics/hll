// This is a Go implementation of the HyperLogLog++ algorithm from "HyperLogLog in Practice:
// Algorithmic Engineering of a State of The Art Cardinality Estimation Algorithm" by Heule,
// Nunkesser and Hall of Google. This is a cardinality estimation algorithm: given a stream of input
// elements, it will estimate the number of unique items in the stream. The estimation error can be
// controlled by choosing how much memory to use. HyperLogLog++ improves on the basic HyperLogLog
// algorithm by using less space, improving accuracy, and correcting bias.
//
// This code is a translation of the pseudocode contained in Figures 6 and 7 of the Google paper.
// Not all algorithms are provided in the paper, but we've tried our best to be true to the authors'
// intent when writing the omitted algorithms. We're not trying to be creative, we're just
// implementing the algorithm described in the paper as directly as possible.
//
// The HyperLogLog++ paper is available at
// http://static.googleusercontent.com/media/research.google.com/en/us/pubs/archive/40671.pdf
package hll
