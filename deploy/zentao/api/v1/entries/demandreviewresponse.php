<?php
/* Normalize native controller results without treating missing/failed results as success. */
function demandReviewResponse($entry, $data)
{
    if(!is_object($data)) return $entry->sendError(502, 'Invalid demand controller response');

    $status = isset($data->status) ? $data->status : null;
    $result = isset($data->result) ? $data->result : null;
    if(in_array($status, array('fail', 'error'), true) || in_array($result, array('fail', 'error'), true))
    {
        return $entry->sendError(400, isset($data->message) ? $data->message : 'Demand action failed');
    }
    if($status !== 'success' && $result !== 'success')
    {
        return $entry->sendError(502, 'Invalid demand controller response');
    }

    // entry::send performs JSON encoding; passing encoded JSON produces a string, not an object.
    return $entry->send(200, array('success' => true, 'message' => isset($data->message) ? $data->message : 'Success'));
}
