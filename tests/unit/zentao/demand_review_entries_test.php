<?php
/* Isolated API adapter tests: no ZenTao bootstrap, network or database. */
class entry
{
    public $response;
    public $allowed = true;
    public $called = false;
    public $method;
    public function batchSetPost($fields) {}
    public function loadController($module, $method)
    {
        if(!$this->allowed) throw new RuntimeException('denied');
        $this->method = $method;
        ob_start();
        return $this;
    }
    public function review($id) { $this->emitNative(); }
    public function submit($id) { $this->emitNative(); }
    public function withdrawReview($id) { $this->emitNative(); }
    public function emitNative()
    {
        $this->called = true;
        echo json_encode($this->response);
        $output = ob_get_clean();
        echo $output;
    }
    public function getData() { return json_decode(ob_get_clean()); }
    public function sendError($code, $error) { return array($code, array('error' => $error)); }
    public function send($code, $data) { return array($code, $data); }
}
function verify($value, $message)
{
    if(!$value) throw new RuntimeException($message);
}
$root = __DIR__ . '/../../../deploy/zentao/api/v1/entries/';
foreach(array('demandreview', 'demandwithdrawreview', 'demandsubmitreview') as $file) require $root . $file . '.php';
$cases = array(
    array((object)array('result' => 'success'), 200),
    array((object)array('status' => 'success'), 200),
    array((object)array('result' => 'fail', 'message' => 'denied'), 400),
    array((object)array('status' => 'error', 'message' => (object)array('reviewer' => array('required'))), 400),
    array((object)array('status' => 'success', 'result' => 'fail'), 400),
    array((object)array('message' => 'unknown'), 502),
    array(null, 502)
);
foreach(array('demandReviewEntry' => 'review', 'demandWithdrawReviewEntry' => 'withdrawReview', 'demandSubmitReviewEntry' => 'submit') as $class => $method)
{
    foreach($cases as $case)
    {
        $entry = new $class();
        $entry->response = $case[0];
        $reply = $entry->post(1);
        verify($entry->called && $entry->method === $method, $class . ' uses actual controller permission');
        verify($reply[0] === $case[1], $class . ' response status');
        verify(is_array($reply[1]), 'response must not be double encoded');
        if($reply[0] === 400 && isset($case[0]->message)) verify($reply[1]['error'] == $case[0]->message, 'preserve validation detail');
    }
    $entry = new $class();
    $entry->allowed = false;
    try { $entry->post(1); throw new RuntimeException('permission bypass'); }
    catch(RuntimeException $e) { verify($e->getMessage() === 'denied' && !$entry->called, 'retain permission check'); }
}
echo "PASS: all three demand adapters, success/error/malformed response and denied permissions\n";
