function fz
  nohup fuzzy-clone cache update >/dev/null 2>&1 &
  disown

  set choice (fuzzy-clone)

  if test $status -ne 0
    return $status
  end

  cd (echo "$choice" | tail -n 1)
end
