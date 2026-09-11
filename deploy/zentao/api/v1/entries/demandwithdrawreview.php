<?php
require_once __DIR__ . "/demandreviewresponse.php";
class demandWithdrawReviewEntry extends entry
{
    public function post($demandID)
    {
        $fields = "comment";
        $this->batchSetPost($fields);

        $control = $this->loadController("demand", "withdrawReview");
        // Native API control::send consumes one output buffer before printing its JSON.
        ob_start();
        $control->withdrawReview($demandID);

        return demandReviewResponse($this, $this->getData());
    }
}
