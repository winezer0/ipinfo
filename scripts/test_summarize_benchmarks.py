import copy
import unittest
import summarize_benchmarks as summary


def fixture():
    return {"kind": "api", "go": "test", "os": "test", "arch": "test", "gomaxprocs": 4, "clock": "test",
            "results": [dict(pool=pool, workers=4, batch=1, endpoint="/ip", requests=100,
                             requests_per_second=100*pool+offset, p95_us=100/pool,
                             errors=0, error_rate=0) for pool in (1, 4) for offset in (-10, 0, 10)]}


class SummaryTests(unittest.TestCase):
    def test_medians_variation_and_pool_gain(self):
        result = summary.summarize([fixture()], 3)
        self.assertEqual(result["rows"][0]["metrics"]["requests_per_second"], {"median": 100, "min": 90, "max": 110})
        self.assertEqual(result["pool_comparisons"][0]["throughput_ratio"], 4)
        self.assertEqual(result["pool_comparisons"][0]["p95_change_percent"], -75)

    def test_baseline_comparison(self):
        before = summary.summarize([fixture()], 3)
        raw = fixture()
        for row in raw["results"]:
            row["requests_per_second"] *= 2
        after = summary.summarize([raw], 3)
        summary.add_baseline(after, before)
        self.assertEqual(after["baseline_comparisons"][0]["throughput_ratio"], 2)
        before["environment"]["clock"] = "different"
        with self.assertRaises(ValueError):
            summary.add_baseline(after, before)

    def test_rejects_mixed_or_incomplete_samples(self):
        raw = fixture()
        other = copy.deepcopy(raw)
        other["arch"] = "different"
        with self.assertRaises(ValueError):
            summary.summarize([raw, other], 3)
        with self.assertRaises(ValueError):
            summary.summarize([raw], 4)
        raw["results"][0]["p95_us"] = float("nan")
        with self.assertRaises(ValueError):
            summary.summarize([raw], 3)

    def test_failures_are_visible(self):
        raw = fixture()
        raw["results"][0]["errors"] = 1
        result = summary.summarize([raw], 3)
        self.assertEqual(result["rows"][0]["failed_samples"], 1)
        self.assertIn("failed_samples=1", summary.render(result))


if __name__ == "__main__":
    unittest.main()
