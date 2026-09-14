// Local-only workerd Worker: evaluates kit/i18n/intltest cases with JavaScript's Intl (intl.mjs) and returns
// {meta, results: {id: output}}.
import cases from "./cases.json";
import { evaluate } from "./intl.mjs";

export default {
  async fetch() {
    return Response.json(evaluate(cases));
  },
};
