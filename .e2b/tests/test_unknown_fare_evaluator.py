import importlib.util
import unittest
from pathlib import Path

SOURCE = Path(__file__).resolve().parents[2] / '.relai/harbor/agent-tests/playground-unknown-fare/tests/evaluate.py'
spec = importlib.util.spec_from_file_location('unknown_fare_evaluator', SOURCE)
evaluator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(evaluator)


def trace(fare='Flex Saver Plus', result=evaluator.STANDARD_FALLBACK, calls=True):
    return {'schema_version': 'ATIF-v1.7', 'agent': {'name': 'test', 'version': '1'}, 'steps': [
        {'step_id': 1, 'source': 'user', 'message': 'My ticket says Flex Saver Plus. How many checked bags are included?'},
        {'step_id': 2, 'source': 'agent', 'tool_calls': ([{'tool_call_id': 'call', 'function_name': 'baggage_policy', 'arguments': {'ticket_type': fare}}] if calls else []),
         'observation': {'results': [{'source_call_id': 'call', 'content': result}]}}
    ]}


class GroundingTest(unittest.TestCase):
    def test_original_fallback_fails_even_with_cautious_assistant(self):
        t = trace()
        t['steps'].append({'step_id': 3, 'source': 'agent', 'message': 'I cannot confirm this fare. Please clarify.'})
        scores, feedback = evaluator.evaluate(t)
        self.assertEqual(scores['verify-unknown-fare-fallback'], 0)
        self.assertIn('unrecognized fare', feedback['verify-unknown-fare-fallback'])

    def test_known_fare_is_not_a_false_positive(self):
        self.assertEqual(evaluator.evaluate(trace(fare='standard'))[0]['verify-unknown-fare-fallback'], 1)

    def test_explicit_unknown_policy_passes(self):
        self.assertEqual(evaluator.evaluate(trace(result='Unknown fare. Ask for clarification.'))[0]['verify-unknown-fare-fallback'], 1)

    def test_missing_result_cannot_establish_grounding(self):
        self.assertEqual(evaluator.evaluate(trace(result=None))[0]['verify-unknown-fare-fallback'], 0)

    def test_no_call_reports_conditional_coverage(self):
        self.assertIn('does not verify the tool repair', evaluator.evaluate(trace(calls=False))[1]['verify-unknown-fare-fallback'])

    def test_corrupt_trace_is_rejected(self):
        t = trace()
        t['steps'][1]['step_id'] = 4
        with self.assertRaises(AssertionError):
            evaluator.evaluate(t)


if __name__ == '__main__':
    unittest.main()
